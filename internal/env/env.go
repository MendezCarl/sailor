// Package env handles variable loading and ${var} interpolation.
// It has no knowledge of HTTP or terminal output.
//
// Variable resolution order (highest to lowest priority):
//  1. CLI --var flags
//  2. OS environment variables
//  3. .env.local in the search directory
//  4. .env in the search directory
//  5. Base vars (e.g. collection base_url field)
//
// All keys are normalised to lowercase so that ${auth_token} in a request
// resolves AUTH_TOKEN from a .env file without any extra configuration.
package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/request"
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
// dir is where .env and .env.local are looked up (typically the directory
// containing the request file or collection being executed).
func Collect(dir string, baseVars Vars, cliVars Vars) (Vars, error) {
	result := Vars{}

	// 1. Base vars (lowest priority — e.g. collection.base_url).
	for k, v := range baseVars {
		result[strings.ToLower(k)] = v
	}

	// 2. .env file.
	dotenv, err := LoadDotEnv(filepath.Join(dir, ".env"))
	if err != nil {
		return nil, err
	}
	for k, v := range dotenv {
		result[k] = v
	}

	// 3. .env.local (overrides .env).
	local, err := LoadDotEnv(filepath.Join(dir, ".env.local"))
	if err != nil {
		return nil, err
	}
	for k, v := range local {
		result[k] = v
	}

	// 4. OS environment variables.
	for _, entry := range os.Environ() {
		idx := strings.IndexByte(entry, '=')
		if idx < 0 {
			continue
		}
		result[strings.ToLower(entry[:idx])] = entry[idx+1:]
	}

	// 5. CLI --var flags (highest priority).
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
