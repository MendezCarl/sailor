// Package env handles variable loading and ${var} interpolation.
// It has no knowledge of HTTP or terminal output.
//
// Variable resolution order (highest to lowest priority):
//  1. CLI --var flags
//  2. OS environment variables
//  3. .env.local in the search directory
//  4. .env.<envName> in the search directory (only when an env is active)
//  5. .env in the search directory
//  6. Active environment YAML file (.apitool/envs/)
//  7. Base vars (e.g. collection base_url field)
//
// All keys are normalised to lowercase so that ${auth_token} in a request
// resolves AUTH_TOKEN from a .env file without any extra configuration.
package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/request"
	"gopkg.in/yaml.v3"
)

// Vars is a map of variable name → value.
// Keys are always lowercase.
type Vars map[string]string

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

// Collect builds a merged Vars from all sources for a given directory.
// dir is where .env files are looked up (typically the directory containing
// the request file or collection being executed).
// envFilePath and envName identify an optional active environment YAML file;
// pass empty strings to skip that layer.
func Collect(dir string, baseVars Vars, cliVars Vars, envFilePath string, envName string) (Vars, error) {
	result := Vars{}

	// 1. Base vars (lowest priority — e.g. collection.base_url).
	for k, v := range baseVars {
		result[strings.ToLower(k)] = v
	}

	// 2. Environment YAML file (above base, below .env files).
	if envFilePath != "" {
		envVars, err := LoadEnvFile(envFilePath, envName)
		if err != nil {
			return nil, err
		}
		for k, v := range envVars {
			result[k] = v
		}
	}

	// 3. .env file.
	dotenv, err := LoadDotEnv(filepath.Join(dir, ".env"))
	if err != nil {
		return nil, err
	}
	for k, v := range dotenv {
		result[k] = v
	}

	// 4. .env.<envName> (environment-specific overlay, only when env is active).
	if envName != "" {
		envDot, err := LoadDotEnv(filepath.Join(dir, ".env."+envName))
		if err != nil {
			return nil, err
		}
		for k, v := range envDot {
			result[k] = v
		}
	}

	// 5. .env.local (overrides all .env files).
	local, err := LoadDotEnv(filepath.Join(dir, ".env.local"))
	if err != nil {
		return nil, err
	}
	for k, v := range local {
		result[k] = v
	}

	// 6. OS environment variables.
	for _, entry := range os.Environ() {
		idx := strings.IndexByte(entry, '=')
		if idx < 0 {
			continue
		}
		result[strings.ToLower(entry[:idx])] = entry[idx+1:]
	}

	// 7. CLI --var flags (highest priority).
	for k, v := range cliVars {
		result[strings.ToLower(k)] = v
	}

	return result, nil
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

// Interpolate replaces all ${varname} references in s using vars.
// Returns the result string and a slice of any variable names that were
// not found in vars (so callers can warn the user).
func Interpolate(s string, vars Vars) (string, []string) {
	if !strings.Contains(s, "${") {
		return s, nil // fast path: nothing to interpolate
	}

	var result strings.Builder
	var undefined []string

	i := 0
	for i < len(s) {
		start := strings.Index(s[i:], "${")
		if start < 0 {
			result.WriteString(s[i:])
			break
		}
		start += i

		result.WriteString(s[i:start]) // text before the ${

		end := strings.Index(s[start:], "}")
		if end < 0 {
			// Unclosed ${: write the rest verbatim.
			result.WriteString(s[start:])
			break
		}
		end += start

		varName := s[start+2 : end]
		key := strings.ToLower(varName)

		if val, ok := vars[key]; ok {
			result.WriteString(val)
		} else {
			result.WriteString(s[start : end+1]) // preserve literal ${varname}
			undefined = append(undefined, varName)
		}

		i = end + 1
	}

	return result.String(), undefined
}

// ParseVarFlag parses a "key=value" string from a --var flag.
func ParseVarFlag(s string) (string, string, error) {
	idx := strings.IndexByte(s, '=')
	if idx < 0 {
		return "", "", fmt.Errorf("--var %q: expected key=value format", s)
	}
	return strings.TrimSpace(s[:idx]), s[idx+1:], nil
}

// Apply returns a copy of req with all ${var} references resolved using vars.
// Undefined variable references are preserved verbatim and a warning is
// written to stderr for each unique undefined name.
func Apply(req *request.Request, vars Vars) *request.Request {
	out := &request.Request{
		Name:   req.Name,
		Method: req.Method,
	}

	var allUndefined []string

	var urlUndef []string
	out.URL, urlUndef = Interpolate(req.URL, vars)
	allUndefined = append(allUndefined, urlUndef...)

	if req.Body != "" {
		var u []string
		out.Body, u = Interpolate(req.Body, vars)
		allUndefined = append(allUndefined, u...)
	}

	if len(req.Headers) > 0 {
		out.Headers = make(map[string]string, len(req.Headers))
		for k, v := range req.Headers {
			interpolated, u := Interpolate(v, vars)
			out.Headers[k] = interpolated
			allUndefined = append(allUndefined, u...)
		}
	}

	if len(req.Params) > 0 {
		out.Params = make(map[string]string, len(req.Params))
		for k, v := range req.Params {
			interpolated, u := Interpolate(v, vars)
			out.Params[k] = interpolated
			allUndefined = append(allUndefined, u...)
		}
	}

	// Warn once per unique undefined variable name.
	seen := make(map[string]bool)
	for _, name := range allUndefined {
		if !seen[name] {
			fmt.Fprintf(os.Stderr, "warning: undefined variable ${%s}\n", name)
			seen[name] = true
		}
	}

	return out
}
