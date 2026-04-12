package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadDotEnv parses a .env-style file.
// Missing files are silently ignored (not an error).
// Lines starting with # are comments. Blank lines are skipped.
// Values may be wrapped in single or double quotes; quotes are stripped.
func LoadDotEnv(path string) (Vars, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Vars{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}
	defer f.Close()

	vars := Vars{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue // not key=value, skip tolerantly
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip optional surrounding quotes.
		if len(val) >= 2 {
			q := val[0]
			if (q == '"' || q == '\'') && val[len(val)-1] == q {
				val = val[1 : len(val)-1]
			}
		}

		vars[strings.ToLower(key)] = val
	}

	return vars, scanner.Err()
}

// envFileProbe is used to detect whether a YAML file is in multi-environment
// format. If Environments is non-nil after unmarshaling, it is multi-env.
type envFileProbe struct {
	Environments map[string]map[string]string `yaml:"environments"`
}

// LoadEnvFile loads a YAML environment file and returns the variables for the
// named environment. Two formats are supported:
//
// Single-environment: the whole file is a flat key→value map. envName is
// ignored. schema_version is excluded from the returned vars.
//
// Multi-environment: variables are nested under an "environments" key with
// named sub-maps. envName selects which environment to load.
//
// Missing files are silently ignored (same behaviour as LoadDotEnv).
func LoadEnvFile(path string, envName string) (Vars, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Vars{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}

	// Detect format.
	var probe envFileProbe
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filepath.Base(path), err)
	}

	if probe.Environments != nil {
		// Multi-environment format.
		if envName == "" {
			return nil, fmt.Errorf("environment name required for multi-environment file %s", filepath.Base(path))
		}
		env, ok := probe.Environments[envName]
		if !ok {
			names := make([]string, 0, len(probe.Environments))
			for n := range probe.Environments {
				names = append(names, n)
			}
			sort.Strings(names)
			return nil, fmt.Errorf("environment %q not found in %s; available: %s", envName, filepath.Base(path), strings.Join(names, ", "))
		}
		vars := make(Vars, len(env))
		for k, v := range env {
			vars[strings.ToLower(k)] = v
		}
		return vars, nil
	}

	// Single-environment format: unmarshal into a generic map to capture all keys.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filepath.Base(path), err)
	}

	vars := make(Vars, len(raw))
	for k, v := range raw {
		key := strings.ToLower(k)
		if key == "schema_version" {
			continue // metadata field, not a user variable
		}
		vars[key] = fmt.Sprintf("%v", v)
	}
	return vars, nil
}

// EnvEntry describes one discoverable environment.
type EnvEntry struct {
	Name   string // environment name
	Source string // file path it was found in
	Multi  bool   // true if sourced from a multi-environment file
}

// ListEnvs returns all environments discoverable from cwd.
// It scans the project-local directory (.apitool/envs/) and the global
// directory (~/.config/apitool/envs/).
//
// Single-env yaml files (e.g. staging.yaml) contribute one entry whose name
// is the filename without the .yaml extension.
// Multi-env files (environments.yaml) contribute one entry per named environment.
// environments.yaml is always treated as multi-env; other *.yaml files are
// treated as single-env.
func ListEnvs(cwd string) ([]EnvEntry, error) {
	home, _ := os.UserHomeDir()

	dirs := []string{filepath.Join(cwd, ".apitool", "envs")}
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".config", "apitool", "envs"))
	}

	seen := map[string]bool{}
	var entries []EnvEntry

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			return nil, fmt.Errorf("listing envs in %s: %w", dir, err)
		}

		for _, path := range files {
			base := filepath.Base(path)

			if base == "environments.yaml" {
				// Multi-env file: extract environment names.
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				var probe envFileProbe
				if err := yaml.Unmarshal(data, &probe); err != nil {
					continue
				}
				for name := range probe.Environments {
					key := name + "|" + path
					if seen[key] {
						continue
					}
					seen[key] = true
					entries = append(entries, EnvEntry{Name: name, Source: path, Multi: true})
				}
				continue
			}

			// Single-env file: name is filename without extension.
			name := strings.TrimSuffix(base, ".yaml")
			key := name + "|" + path
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, EnvEntry{Name: name, Source: path, Multi: false})
		}
	}

	return entries, nil
}

// ResolveEnvFile searches for the environment YAML file for envName.
// It checks the project directory (.apitool/envs/) before the global config
// directory (~/.config/apitool/envs/). Returns the file path and the
// envName to pass to LoadEnvFile (empty string for single-env files).
// Returns ("", "") if envName is empty or no env file is found.
func ResolveEnvFile(cwd string, envName string) (filePath string, resolvedEnvName string) {
	if envName == "" {
		return "", ""
	}

	home, _ := os.UserHomeDir()

	candidates := []struct {
		path    string
		envName string
	}{
		{filepath.Join(cwd, ".apitool", "envs", envName+".yaml"), ""},
		{filepath.Join(cwd, ".apitool", "envs", "environments.yaml"), envName},
	}
	if home != "" {
		candidates = append(candidates,
			struct {
				path    string
				envName string
			}{filepath.Join(home, ".config", "apitool", "envs", envName+".yaml"), ""},
			struct {
				path    string
				envName string
			}{filepath.Join(home, ".config", "apitool", "envs", "environments.yaml"), envName},
		)
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.path); err == nil {
			return c.path, c.envName
		}
	}
	return "", ""
}
