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
	"fmt"
	"io"
	"os"

	"github.com/MendezCarl/sailor.git/internal/request"
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

	// NoPager disables automatic paging through $PAGER even when stdout is
	// a TTY and the response body is large. Has no effect in raw or json mode.
	NoPager bool
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		Format: "pretty",
		Color:  "auto",
	}
}

// resolveColor decides whether to emit ANSI color codes.
// Respects the NO_COLOR convention (https://no-color.org/): any non-empty
// value disables color regardless of mode.
func resolveColor(mode string, w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto" and anything unrecognised
		return isTTY(w)
	}
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

	// Non-TTY auto-decoration disable.
	// When color is "auto" and the body writer is not a terminal (e.g., output
	// is being piped or redirected), suppress all decoration and emit raw body
	// bytes only. This matches the behavior of curl/httpie when piped.
	if opts.Color == "auto" && !isTTY(body) {
		body.Write(resp.Body) //nolint:errcheck
		return
	}

	if !opts.Quiet {
		color := resolveColor(opts.Color, diag)
		printStatusLine(diag, resp, color)
		if opts.ShowHeaders {
			printHeaders(diag, resp, color)
			fmt.Fprintln(diag) // blank line between headers and body
		}
	}

	// Long response paging: pipe body through $PAGER when active.
	w := body
	if pw, ok := openPager(body, opts.NoPager); ok {
		w = pw
		defer pw.Close()
	}

	printBody(w, resp)
}
