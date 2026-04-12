package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
