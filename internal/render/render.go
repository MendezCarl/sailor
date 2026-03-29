// Package render formats and writes HTTP responses to the terminal.
// It has no knowledge of HTTP, YAML, or file formats.
//
// Output contract (matches docs/architecture.md §10):
//   - Status line  → diag (stderr)
//   - Headers      → diag (stderr), only when showHeaders is true
//   - Body         → body (stdout)
//
// Passing io.Writer arguments instead of using os.Stdout/Stderr directly
// allows this package to be tested without process-level output capture.
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/MendezCarl/sailor.git/internal/request"
)

// ANSI color codes. Applied only when the diagnostic writer is a TTY.
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// Options controls how a Response is rendered.
type Options struct {
	// Format is "pretty" (default), "raw", or "json".
	//   pretty: status line + formatted body (default)
	//   raw:    body bytes only, verbatim, no decoration
	//   json:   full response as a JSON object to stdout
	Format string

	// Color is "auto" (default), "always", or "never".
	// "auto" enables ANSI color only when diag is a TTY.
	Color string

	// ShowHeaders prints response headers before the body (pretty mode only).
	ShowHeaders bool

	// Quiet suppresses the status line and all header output on stderr.
	// The response body is still written to stdout, formatted as usual.
	// Has no effect in raw or json mode.
	Quiet bool
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		Format: "pretty",
		Color:  "auto",
	}
}

// resolveColor decides whether to emit ANSI color codes.
func resolveColor(mode string, w io.Writer) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto" and anything unrecognised
		return isTTY(w)
	}
}

// Print writes a Response to body and diag according to opts.
//
//   - Format "raw":    body bytes only, verbatim, to body writer
//   - Format "json":   full response as JSON to body writer; diag receives nothing
//   - Format "pretty": status line (+ optional headers) to diag; body to body
//     If Quiet is true, the status line and headers are suppressed.
func Print(body io.Writer, diag io.Writer, resp *request.Response, opts Options) {
	switch opts.Format {
	case "raw":
		body.Write(resp.Body) //nolint:errcheck // best-effort write to stdout
		return
	case "json":
		printJSON(body, resp)
		return
	}

	// pretty mode
	if !opts.Quiet {
		color := resolveColor(opts.Color, diag)
		printStatusLine(diag, resp, color)
		if opts.ShowHeaders {
			printHeaders(diag, resp, color)
			fmt.Fprintln(diag) // blank line between headers and body
		}
	}

	printBody(body, resp)
}

// jsonResponse is the structure written to stdout when Format is "json".
type jsonResponse struct {
	StatusCode int                 `json:"status_code"`
	Status     string              `json:"status"`
	Proto      string              `json:"proto"`
	DurationMS int64               `json:"duration_ms"`
	Size       int                 `json:"size"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
}

// printJSON writes the full response as an indented JSON object to w.
// All output goes to w (the body writer); diag receives nothing.
func printJSON(w io.Writer, resp *request.Response) {
	out := jsonResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Proto:      resp.Proto,
		DurationMS: resp.Duration.Milliseconds(),
		Size:       len(resp.Body),
		Headers:    resp.Headers,
		Body:       string(resp.Body),
	}

	raw, err := json.Marshal(out)
	if err != nil {
		// Should never happen with this concrete struct.
		fmt.Fprintf(w, `{"error":"failed to serialize response"}`+"\n")
		return
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		w.Write(raw) //nolint:errcheck
	} else {
		w.Write(buf.Bytes()) //nolint:errcheck
	}
	fmt.Fprintln(w)
}

// printStatusLine writes the one-line response summary to diag.
// Format: HTTP/1.1 200 OK  (143ms  1.2 KB)
func printStatusLine(w io.Writer, resp *request.Response, color bool) {
	proto := resp.Proto
	if proto == "" {
		proto = "HTTP"
	}

	status := resp.Status
	dur := formatDuration(resp.Duration)
	size := formatSize(len(resp.Body))

	if color {
		sc := statusColor(resp.StatusCode)
		fmt.Fprintf(w, "%s%s %s%s%s  (%s  %s)\n",
			colorBold, proto,
			sc, status, colorReset,
			dur, size,
		)
	} else {
		fmt.Fprintf(w, "%s %s  (%s  %s)\n", proto, status, dur, size)
	}
}

// printHeaders writes response headers to w, sorted for consistent output.
func printHeaders(w io.Writer, resp *request.Response, color bool) {
	// Collect and sort header names so output is deterministic.
	names := make([]string, 0, len(resp.Headers))
	for name := range resp.Headers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		for _, val := range resp.Headers[name] {
			if color {
				fmt.Fprintf(w, "%s%s%s: %s\n", colorCyan, name, colorReset, val)
			} else {
				fmt.Fprintf(w, "%s: %s\n", name, val)
			}
		}
	}
}

// printBody writes the response body to w.
// If the response Content-Type indicates JSON, the body is pretty-printed.
// If pretty-printing fails (malformed JSON), the raw body is written instead.
func printBody(w io.Writer, resp *request.Response) {
	if len(resp.Body) == 0 {
		return
	}

	if isJSON(resp) {
		pretty, err := prettyJSON(resp.Body)
		if err == nil {
			w.Write(pretty) //nolint:errcheck
			// Ensure a trailing newline.
			if len(pretty) > 0 && pretty[len(pretty)-1] != '\n' {
				fmt.Fprintln(w)
			}
			return
		}
		// Malformed JSON: fall through to raw output.
	}

	w.Write(resp.Body) //nolint:errcheck
	// Ensure a trailing newline so the shell prompt appears on its own line.
	if len(resp.Body) > 0 && resp.Body[len(resp.Body)-1] != '\n' {
		fmt.Fprintln(w)
	}
}

// isJSON returns true if the response Content-Type indicates JSON.
func isJSON(resp *request.Response) bool {
	ct := resp.Headers["Content-Type"]
	if len(ct) == 0 {
		return false
	}
	return strings.Contains(strings.ToLower(ct[0]), "json")
}

// prettyJSON formats a JSON byte slice with two-space indentation.
func prettyJSON(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, src, "", "  "); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isTTY returns true if w is an *os.File connected to a terminal.
// Used to decide whether to emit ANSI color codes.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// statusColor returns an ANSI color code based on the HTTP status class.
func statusColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return colorGreen
	case code >= 300 && code < 400:
		return colorCyan
	case code >= 400 && code < 500:
		return colorYellow
	case code >= 500:
		return colorRed
	default:
		return colorReset
	}
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// formatSize returns a human-readable byte size string.
func formatSize(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}
