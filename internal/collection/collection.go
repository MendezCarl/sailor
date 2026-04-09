// Package collection loads and queries YAML collection files.
// It has no knowledge of HTTP execution, terminal output, or variable
// interpolation — those are handled by other packages.
package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/request"
	"gopkg.in/yaml.v3"
)

// Folder groups related requests within a collection.
// Folders may be nested to any depth via the Folders field.
type Folder struct {
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Requests    []*request.Request `yaml:"requests"`
	Folders     []*Folder          `yaml:"folders"`
}

// Collection represents a named group of saved requests loaded from a YAML file.
type Collection struct {
	SchemaVersion int                `yaml:"schema_version"`
	Name          string             `yaml:"name"`
	Description   string             `yaml:"description"`
	BaseURL       string             `yaml:"base_url"`
	Requests      []*request.Request `yaml:"requests"`
	Folders       []*Folder          `yaml:"folders"`
}

// CollectionEntry pairs a loaded Collection with the file it came from.
type CollectionEntry struct {
	Collection *Collection
	FilePath   string
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

	normalizeMethods(c.Requests, c.Folders)

	return &c, nil
}

// normalizeMethods uppercases HTTP methods for all requests at this level and
// recursively for all nested folders.
func normalizeMethods(reqs []*request.Request, folders []*Folder) {
	for _, req := range reqs {
		req.Method = strings.ToUpper(strings.TrimSpace(req.Method))
	}
	for _, f := range folders {
		normalizeMethods(f.Requests, f.Folders)
	}
}

// FindRequest returns the request identified by target in collection c.
//
// target may be a plain name ("List Users") for top-level requests, or a
// dot-separated path ("Users.List Users") for requests inside folders.
// Dots within a name component are escaped with a backslash: "Admin\.V2.List".
//
// On failure the error message lists all available request names.
func FindRequest(c *Collection, target string) (*request.Request, error) {
	parts := splitDotPath(target)

	if len(parts) == 1 {
		// Plain name: search top-level requests only.
		for _, req := range c.Requests {
			if req.Name == target {
				return req, nil
			}
		}
	} else {
		// Dot-path: navigate folders then find the request.
		req := findInFolders(c.Folders, parts)
		if req != nil {
			return req, nil
		}
	}

	names := allRequestNames(c)
	if len(names) == 0 {
		return nil, fmt.Errorf("request %q not found (collection %q has no requests)", target, c.Name)
	}
	listed := make([]string, len(names))
	for i, n := range names {
		listed[i] = "  " + n
	}
	return nil, fmt.Errorf("request %q not found in collection %q\n\navailable requests:\n%s",
		target, c.Name, strings.Join(listed, "\n"))
}

// findInFolders walks folders matching the path components, returning the
// request named by the last component. Returns nil if not found.
func findInFolders(folders []*Folder, parts []string) *request.Request {
	if len(parts) < 2 {
		return nil
	}
	folderName := parts[0]
	rest := parts[1:]

	for _, f := range folders {
		if f.Name != folderName {
			continue
		}
		if len(rest) == 1 {
			// Last component: find request by name.
			for _, req := range f.Requests {
				if req.Name == rest[0] {
					return req
				}
			}
			return nil
		}
		// More path components: recurse into sub-folders.
		return findInFolders(f.Folders, rest)
	}
	return nil
}

// splitDotPath splits a dot-separated path into components.
// A backslash before a dot escapes it: "Admin\.V2.List" → ["Admin.V2", "List"].
func splitDotPath(s string) []string {
	var parts []string
	var current strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '.' {
			current.WriteByte('.')
			i++ // skip the escaped dot
			continue
		}
		if s[i] == '.' {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(s[i])
	}
	parts = append(parts, current.String())
	return parts
}

// allRequestNames returns the names of all requests in c, including folder
// paths (e.g. "Users.List Users"). Used for error messages and display.
func allRequestNames(c *Collection) []string {
	var names []string
	for _, req := range c.Requests {
		names = append(names, req.Name)
	}
	for _, f := range c.Folders {
		names = append(names, folderRequestNames(f, f.Name)...)
	}
	return names
}

func folderRequestNames(f *Folder, prefix string) []string {
	var names []string
	for _, req := range f.Requests {
		names = append(names, prefix+"."+req.Name)
	}
	for _, sub := range f.Folders {
		names = append(names, folderRequestNames(sub, prefix+"."+sub.Name)...)
	}
	return names
}

// ListAll loads all *.yaml files in dir and returns every collection that
// parses successfully, sorted by collection name.
// Files that fail to parse are silently skipped.
// Returns an error only if the directory cannot be read.
func ListAll(dir string) ([]CollectionEntry, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("listing collections in %s: %w", dir, err)
	}

	var result []CollectionEntry
	for _, path := range entries {
		c, err := LoadFile(path)
		if err != nil {
			continue
		}
		result = append(result, CollectionEntry{Collection: c, FilePath: path})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Collection.Name < result[j].Collection.Name
	})
	return result, nil
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

// CountRequests returns the total number of requests in c, including those
// inside folders at all depths.
func CountRequests(c *Collection) int {
	return len(c.Requests) + countFolderRequests(c.Folders)
}

func countFolderRequests(folders []*Folder) int {
	n := 0
	for _, f := range folders {
		n += len(f.Requests) + countFolderRequests(f.Folders)
	}
	return n
}
