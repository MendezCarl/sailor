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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/request"
)

// Vars is a map of variable name → value.
// Keys are always lowercase.
type Vars map[string]string

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
// Returns an error if the auth block is invalid (e.g. unknown type, missing fields).
func Apply(req *request.Request, vars Vars) (*request.Request, error) {
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

	// Interpolate variable references in auth block fields, then apply auth.
	if req.Auth != nil {
		token, tokenUndef := Interpolate(req.Auth.Token, vars)
		username, usernameUndef := Interpolate(req.Auth.Username, vars)
		password, passwordUndef := Interpolate(req.Auth.Password, vars)
		key, keyUndef := Interpolate(req.Auth.Key, vars)
		allUndefined = append(allUndefined, tokenUndef...)
		allUndefined = append(allUndefined, usernameUndef...)
		allUndefined = append(allUndefined, passwordUndef...)
		allUndefined = append(allUndefined, keyUndef...)
		out.Auth = &request.AuthConfig{
			Type:     req.Auth.Type,
			Token:    token,
			Username: username,
			Password: password,
			Key:      key,
			Header:   req.Auth.Header, // literal header name, not interpolated
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

	return applyAuth(out)
}
