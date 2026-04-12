// Package curl handles converting between curl command strings and the
// internal Request struct. It has no knowledge of HTTP execution, terminal
// output, or file formats.
//
// Import path:  curl string → Parse → *request.Request
// Export path:  *request.Request → Export → curl string
package curl

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/request"
)

// Parse converts a curl command string into a Request.
//
// Returns the request, a list of warnings for unrecognized flags, and an
// error if the input is not a recognizable curl command.
//
// The parser is tolerant: unrecognized flags are skipped with a warning
// rather than causing a hard failure. This allows curl commands copied from
// documentation or browser DevTools to import cleanly even when they contain
// flags not relevant to the request definition.
func Parse(curlStr string) (*request.Request, []string, error) {
	tokens := tokenize(curlStr)
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("empty curl command")
	}

	// The first token must be "curl".
	if tokens[0] != "curl" {
		return nil, nil, fmt.Errorf("input does not start with \"curl\"")
	}

	var (
		method   string
		rawURL   string
		headers  = map[string]string{}
		body     string
		warnings []string
	)

	i := 1
	for i < len(tokens) {
		tok := tokens[i]

		// Single-char flags may be combined: -sL, -sk, etc.
		// We handle them one at a time so we do not need to unpack bundles.

		switch tok {
		case "-X", "--request":
			i++
			if i < len(tokens) {
				method = strings.ToUpper(tokens[i])
			}

		case "-H", "--header":
			i++
			if i < len(tokens) {
				name, val, ok := strings.Cut(tokens[i], ":")
				if ok {
					headers[strings.TrimSpace(name)] = strings.TrimSpace(val)
				}
			}

		case "-d", "--data", "--data-raw", "--data-binary":
			i++
			if i < len(tokens) {
				body = tokens[i]
			}

		case "--json":
			// --json sets the body AND adds Content-Type + Accept headers.
			i++
			if i < len(tokens) {
				body = tokens[i]
				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/json"
				}
				if _, ok := headers["Accept"]; !ok {
					headers["Accept"] = "application/json"
				}
			}

		case "-u", "--user":
			i++
			if i < len(tokens) {
				encoded := base64.StdEncoding.EncodeToString([]byte(tokens[i]))
				headers["Authorization"] = "Basic " + encoded
			}

		case "-b", "--cookie":
			i++
			if i < len(tokens) {
				headers["Cookie"] = tokens[i]
			}

		case "--url":
			i++
			if i < len(tokens) {
				rawURL = tokens[i]
			}

		// Flags we recognize and silently ignore (no semantic relevance to
		// the saved request definition).
		case "-L", "--location",
			"-k", "--insecure",
			"-s", "--silent",
			"-S", "--show-error",
			"-v", "--verbose",
			"-f", "--fail",
			"--compressed",
			"--http1.0", "--http1.1", "--http2", "--http2-prior-knowledge",
			"-I", "--head":
			// no-arg flags: skip

		case "-o", "--output",
			"--max-redirs",
			"--connect-timeout",
			"-m", "--max-time",
			"-e", "--referer",
			"-A", "--user-agent",
			"-c", "--cookie-jar",
			"--proxy", "-x",
			"--cacert", "--cert", "--key",
			"--resolve":
			// one-arg flags: skip flag and its argument
			i++

		default:
			if strings.HasPrefix(tok, "-") {
				// Unknown flag: skip it (plus its argument if it looks like a
				// value-taking flag) and warn.
				//
				// Heuristic: if the next token does not start with "-" and is
				// not the URL candidate (i.e. starts with "http"), treat it as
				// the flag's argument and skip it too.
				if !isKnownNoArgFlag(tok) && i+1 < len(tokens) &&
					!strings.HasPrefix(tokens[i+1], "-") &&
					!strings.HasPrefix(tokens[i+1], "http") {
					i++
				}
				warnings = append(warnings, fmt.Sprintf("unsupported flag %q ignored", tok))
			} else {
				// Positional argument: the URL.
				if rawURL == "" {
					rawURL = tok
				}
			}
		}

		i++
	}

	if rawURL == "" {
		return nil, warnings, fmt.Errorf("no URL found in curl command")
	}

	// Extract query parameters from the URL into Params, and keep the clean
	// base URL without the query string.
	baseURL, params, err := splitURL(rawURL)
	if err != nil {
		// URL is malformed but we can still proceed with the raw string.
		baseURL = rawURL
	}

	// Determine method defaults.
	if method == "" {
		if body != "" {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	req := &request.Request{
		Name:   "Imported Request",
		Method: method,
		URL:    baseURL,
	}
	if len(headers) > 0 {
		req.Headers = headers
	}
	if len(params) > 0 {
		req.Params = params
	}
	if body != "" {
		req.Body = body
	}

	return req, warnings, nil
}

// splitURL separates a URL's query string into a params map.
// Returns the base URL (scheme + host + path) and the query parameters.
func splitURL(rawURL string) (string, map[string]string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, nil, err
	}

	q := u.Query()
	if len(q) == 0 {
		return rawURL, nil, nil
	}

	params := make(map[string]string, len(q))
	for k, vals := range q {
		if len(vals) > 0 {
			params[k] = vals[0]
		}
	}

	u.RawQuery = ""
	return u.String(), params, nil
}

// isKnownNoArgFlag returns true for flags that do not take a following value
// argument. Used to decide whether to skip one extra token for unknown flags.
func isKnownNoArgFlag(flag string) bool {
	// We only need a rough heuristic here; unknown flags that do not match
	// will trigger the "skip next token" path, which may occasionally be
	// wrong for truly unknown no-arg flags — but that is acceptable for an
	// import tool. The worst outcome is a garbled URL or extra warning.
	noArgPrefixes := []string{
		"--http1", "--http2", "--compressed",
		"--location", "--insecure", "--silent", "--verbose",
		"--fail", "--head", "--show-error",
	}
	for _, prefix := range noArgPrefixes {
		if flag == prefix || strings.HasPrefix(flag, prefix) {
			return true
		}
	}
	return false
}

