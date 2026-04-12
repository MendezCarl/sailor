// Package request defines the canonical internal representation of an HTTP
// request and response. All input paths (YAML files, future CLI flags, cURL
// import) normalize to Request before execution.
package request

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AuthConfig is an optional convenience block for constructing authentication
// headers. When present, the executor translates it into the appropriate
// Authorization header before sending. An explicit Authorization header in
// Headers always takes precedence over the auth block.
type AuthConfig struct {
	// Type is the auth scheme. Valid values: "bearer", "basic", "apikey".
	Type string `yaml:"type"`
	// Token is used for bearer auth: Authorization: Bearer <token>.
	Token string `yaml:"token,omitempty"`
	// Username and Password are used for basic auth.
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	// Key is the value for apikey auth.
	Key string `yaml:"key,omitempty"`
	// Header is the header name for apikey auth (default: "Authorization").
	Header string `yaml:"header,omitempty"`
}

// Request is the internal representation of an HTTP request.
// It is populated from a YAML file by LoadFile, and consumed by the executor.
type Request struct {
	SchemaVersion int               `yaml:"schema_version"`
	Name          string            `yaml:"name"`
	Method        string            `yaml:"method"`
	URL           string            `yaml:"url"`
	Headers       map[string]string `yaml:"headers"`
	Params        map[string]string `yaml:"params"`
	Body          string            `yaml:"body"`
	// Timeout overrides the global config timeout for this specific request.
	// Accepts Go duration strings: "30s", "1m", "90s". Empty means use the
	// global config value.
	Timeout string      `yaml:"timeout,omitempty"`
	Auth    *AuthConfig `yaml:"auth,omitempty"`
}

// Response holds the result of executing a Request.
// It is produced by the executor and consumed by the renderer.
type Response struct {
	StatusCode int
	Status     string // e.g. "200 OK"
	Proto      string // e.g. "HTTP/1.1"
	Headers    map[string][]string
	Body       []byte
	Duration   time.Duration
}

// LoadFile reads a single request YAML file from disk and returns a Request.
// Returns a user-facing error if the file is missing, unreadable, or invalid.
//
// TODO: when collection support is added, collection loading will move to
// internal/collection. This function handles the standalone -f <file> case.
func LoadFile(path string) (*Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("request file not found: %s", path)
	}

	var req Request
	if err := yaml.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid YAML in request file: %w", err)
	}

	if strings.TrimSpace(req.Method) == "" {
		return nil, fmt.Errorf("request file is missing required field: method")
	}
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("request file is missing required field: url")
	}

	req.Method = strings.ToUpper(strings.TrimSpace(req.Method))

	return &req, nil
}
