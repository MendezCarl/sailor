package curl

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/apitool/apitool/internal/request"
)

// Export serializes a Request to a curl command string.
//
// The output uses long flag names for readability and is formatted across
// multiple lines using backslash continuation when the command has headers
// or a body. Values are single-quoted.
//
// Variable references (${base_url}) are preserved as-is; callers are
// expected to resolve them before sharing the output.
func Export(req *request.Request) string {
	var parts []string

	// Build the full URL including any query parameters from req.Params.
	fullURL := buildURL(req.URL, req.Params)

	// Method: omit --request for GET (the curl default) to keep output clean.
	if req.Method != "" && req.Method != "GET" {
		parts = append(parts, fmt.Sprintf("--request %s", req.Method))
	}

	parts = append(parts, fmt.Sprintf("--url %s", singleQuote(fullURL)))

	// Headers, sorted alphabetically for stable, diffable output.
	if len(req.Headers) > 0 {
		names := make([]string, 0, len(req.Headers))
		for name := range req.Headers {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			parts = append(parts, fmt.Sprintf("--header %s", singleQuote(name+": "+req.Headers[name])))
		}
	}

	// Body.
	if req.Body != "" {
		parts = append(parts, fmt.Sprintf("--data %s", singleQuote(req.Body)))
	}

	if len(parts) == 0 {
		// Bare GET with no headers: single line.
		return "curl " + singleQuote(fullURL)
	}

	// Multi-line format with backslash continuation.
	var sb strings.Builder
	sb.WriteString("curl")
	for _, part := range parts {
		sb.WriteString(" \\\n  ")
		sb.WriteString(part)
	}
	return sb.String()
}

// buildURL appends query parameters from params to rawURL.
// Returns rawURL unchanged if params is empty.
func buildURL(rawURL string, params map[string]string) string {
	if len(params) == 0 {
		return rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		// If the URL is unparseable (e.g. contains ${variables}), append
		// params manually rather than failing.
		return appendParamsManually(rawURL, params)
	}

	q := u.Query()
	// Sort keys for stable output.
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q.Set(k, params[k])
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// appendParamsManually constructs a query string without url.Parse so that
// URLs containing template variables (e.g. ${base_url}/path) are handled
// gracefully.
func appendParamsManually(rawURL string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(params[k]))
	}

	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + strings.Join(pairs, "&")
}

// singleQuote wraps a string in single quotes.
// If the string itself contains a single quote, the standard shell idiom
// is used: end the quoted section, emit the literal quote, resume quoting.
//
//	hello'world  →  'hello'"'"'world'
func singleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
