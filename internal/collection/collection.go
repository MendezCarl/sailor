// Package collection loads and queries YAML collection files.
// It has no knowledge of HTTP execution, terminal output, or variable
// interpolation — those are handled by other packages.
package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/request"
	"gopkg.in/yaml.v3"
)

// Collection represents a named group of saved requests loaded from a YAML file.
type Collection struct {
	SchemaVersion int               `yaml:"schema_version"`
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	BaseURL       string            `yaml:"base_url"`
	Requests      []*request.Request `yaml:"requests"`
	// Folders are not implemented in v0.1.
}

// LoadFile reads and parses a collection YAML file.
func LoadFile(path string) (*Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("collection file not found: %s", path)
	}

	var c Collection
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid YAML in collection file: %w", err)
	}

	if c.Name == "" {
		return nil, fmt.Errorf("collection file missing required field: name")
	}

	// Normalise HTTP methods for all requests.
	for _, req := range c.Requests {
		req.Method = strings.ToUpper(strings.TrimSpace(req.Method))
	}

	return &c, nil
}

// FindRequest returns the first request in c whose Name exactly matches target.
// On failure it returns a list of available names to help the user.
func FindRequest(c *Collection, target string) (*request.Request, error) {
	for _, req := range c.Requests {
		if req.Name == target {
			return req, nil
		}
	}

	names := make([]string, 0, len(c.Requests))
	for _, req := range c.Requests {
		names = append(names, "  "+req.Name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("request %q not found (collection %q has no requests)", target, c.Name)
	}
	return nil, fmt.Errorf("request %q not found in collection %q\n\navailable requests:\n%s",
		target, c.Name, strings.Join(names, "\n"))
}

// Search loads all *.yaml files in dir and returns the first collection and
// request whose Name matches target.
// Files that fail to parse are skipped silently.
func Search(dir string, target string) (*Collection, *request.Request, error) {
	// Check whether the directory exists so we can give a clear error instead
	// of silently returning "not found".
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf(
			"collections directory not found: %s\n\nhint: create the directory and add collection YAML files, or use --collection <file>",
			dir,
		)
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, nil, fmt.Errorf("searching collections in %s: %w", dir, err)
	}

	for _, path := range entries {
		c, err := LoadFile(path)
		if err != nil {
			continue
		}
		req, err := FindRequest(c, target)
		if err == nil {
			return c, req, nil
		}
	}

	return nil, nil, fmt.Errorf("request %q not found in any collection in %s", target, dir)
}

// DefaultCollectionDir returns the path to the default collection directory
// (.apitool/collections) relative to dir. It does not verify the directory exists.
func DefaultCollectionDir(dir string) string {
	return filepath.Join(dir, ".apitool", "collections")
}
