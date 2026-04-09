// Package config loads and merges global and per-project configuration files.
//
// Config lookup order (highest to lowest priority):
//  1. Per-project config: .apitool/config.yaml in the current directory
//  2. Global config:      ~/.config/apitool/config.yaml
//  3. Built-in defaults
//
// Unknown keys in config files are silently ignored.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that unmarshals from human-readable strings
// like "30s", "1m", "5m30s". Zero value means "no timeout" / use default.
type Duration struct {
	d time.Duration
}

// Duration returns the underlying time.Duration value.
func (d Duration) Duration() time.Duration { return d.d }

// UnmarshalYAML implements yaml.Unmarshaler.
// Accepts strings like "30s", "1m", "0" and plain integer seconds.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	switch value.Tag {
	case "!!str":
		parsed, err := time.ParseDuration(value.Value)
		if err != nil {
			return fmt.Errorf("invalid duration %q: use a Go duration string like \"30s\" or \"1m\"", value.Value)
		}
		d.d = parsed
		return nil
	case "!!int":
		// Bare integer: interpret as seconds.
		var n int64
		if err := value.Decode(&n); err != nil {
			return err
		}
		d.d = time.Duration(n) * time.Second
		return nil
	default:
		return fmt.Errorf("invalid duration value: expected a string like \"30s\"")
	}
}

// MarshalYAML implements yaml.Marshaler so Duration round-trips correctly.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.d.String(), nil
}

// OutputConfig holds display-related settings.
type OutputConfig struct {
	// Format controls how the response body is displayed.
	// Valid values: "pretty" (default), "raw".
	Format string `yaml:"format"`

	// Color controls ANSI color output.
	// Valid values: "auto" (default), "always", "never".
	Color string `yaml:"color"`

	// ShowHeaders prints response headers before the body when true.
	ShowHeaders bool `yaml:"show_headers"`
}

// Config holds all user-configurable settings.
// Zero values mean "use the default" and are overridden by Defaults().
type Config struct {
	SchemaVersion int `yaml:"schema_version"`

	// Timeout is the maximum time to wait for a response.
	// Zero means no timeout (http.DefaultClient behaviour).
	Timeout Duration `yaml:"timeout"`

	// DefaultCollection is a path to a collection file used by `apitool run`
	// when --collection is not provided. Takes precedence over the .apitool/
	// directory search.
	DefaultCollection string `yaml:"default_collection"`

	// DefaultEnv is the environment name activated when --env is not provided.
	DefaultEnv string `yaml:"default_env"`

	// Output groups display options.
	Output OutputConfig `yaml:"output"`
}

// Defaults returns a Config with all fields set to their documented defaults.
func Defaults() *Config {
	return &Config{
		Timeout:           Duration{d: 30 * time.Second},
		DefaultCollection: "",
		Output: OutputConfig{
			Format:      "pretty",
			Color:       "auto",
			ShowHeaders: false,
		},
	}
}

// Load builds the effective Config for a given working directory by merging:
//  1. Built-in defaults
//  2. Global config   (~/.config/apitool/config.yaml)
//  3. Project config  (<cwd>/.apitool/config.yaml)
//
// Missing files are silently skipped. Parse errors are returned immediately.
func Load(cwd string) (*Config, error) {
	globalPath, _ := globalConfigPath()
	projectPath := findProjectConfig(cwd)
	return loadFromPaths(globalPath, projectPath)
}

// loadFromPaths is the testable core of Load.
// Either path may be empty, in which case that layer is skipped.
func loadFromPaths(globalPath, projectPath string) (*Config, error) {
	cfg := Defaults()

	if globalPath != "" {
		global, err := LoadFile(globalPath)
		if err != nil {
			return nil, fmt.Errorf("global config: %w", err)
		}
		if global != nil {
			cfg = merge(cfg, global)
		}
	}

	if projectPath != "" {
		project, err := LoadFile(projectPath)
		if err != nil {
			return nil, fmt.Errorf("project config: %w", err)
		}
		if project != nil {
			cfg = merge(cfg, project)
		}
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFile reads and parses a single config YAML file.
// Returns nil, nil if the file does not exist.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &cfg, nil
}

// merge returns a new Config where non-zero fields in override replace
// the corresponding fields in base. base is never mutated.
func merge(base, override *Config) *Config {
	out := *base // shallow copy

	if override.Timeout.d != 0 {
		out.Timeout = override.Timeout
	}
	if override.DefaultCollection != "" {
		out.DefaultCollection = override.DefaultCollection
	}
	if override.DefaultEnv != "" {
		out.DefaultEnv = override.DefaultEnv
	}
	if override.Output.Format != "" {
		out.Output.Format = override.Output.Format
	}
	if override.Output.Color != "" {
		out.Output.Color = override.Output.Color
	}
	// ShowHeaders is a bool: true in the override wins; false means "not set".
	if override.Output.ShowHeaders {
		out.Output.ShowHeaders = true
	}

	return &out
}

// validate checks that all Config fields hold valid values.
func validate(cfg *Config) error {
	switch cfg.Output.Format {
	case "pretty", "raw", "json":
		// valid
	default:
		return fmt.Errorf("config: output.format %q is invalid; valid values: \"pretty\", \"raw\", \"json\"", cfg.Output.Format)
	}

	switch cfg.Output.Color {
	case "auto", "always", "never":
		// valid
	default:
		return fmt.Errorf("config: output.color %q is invalid; valid values: \"auto\", \"always\", \"never\"", cfg.Output.Color)
	}

	if cfg.Timeout.d < 0 {
		return fmt.Errorf("config: timeout must be non-negative")
	}

	return nil
}

// globalConfigPath returns the path to the user-level config file.
// Returns ("", false) if the home directory cannot be determined.
func globalConfigPath() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".config", "apitool", "config.yaml"), true
}

// findProjectConfig returns the path to the project-level config file
// (.apitool/config.yaml) relative to dir. It does not verify the file exists.
func findProjectConfig(dir string) string {
	return filepath.Join(dir, ".apitool", "config.yaml")
}
