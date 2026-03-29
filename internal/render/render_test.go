package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MendezCarl/sailor.git/internal/request"
)

// ---- helpers ----------------------------------------------------------------

func makeResp(status int, contentType string, body []byte) *request.Response {
	headers := map[string][]string{}
	if contentType != "" {
		headers["Content-Type"] = []string{contentType}
	}
	return &request.Response{
		StatusCode: status,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		Headers:    headers,
		Body:       body,
		Duration:   42 * time.Millisecond,
	}
}

// ---- resolveColor -----------------------------------------------------------

func TestResolveColor_Always(t *testing.T) {
	// Non-TTY writer but mode "always" → color enabled.
	var buf bytes.Buffer
	if !resolveColor("always", &buf) {
		t.Error("expected color=true for mode \"always\"")
	}
}

func TestResolveColor_Never(t *testing.T) {
	// Even with a potential TTY, "never" → color disabled.
	var buf bytes.Buffer
	if resolveColor("never", &buf) {
		t.Error("expected color=false for mode \"never\"")
	}
}

func TestResolveColor_Auto_NonTTY(t *testing.T) {
	// bytes.Buffer is not a TTY → auto should return false.
	var buf bytes.Buffer
	if resolveColor("auto", &buf) {
		t.Error("expected color=false for auto + non-TTY writer")
	}
}

func TestResolveColor_UnknownMode_NonTTY(t *testing.T) {
	// Unknown mode falls through to auto behaviour.
	var buf bytes.Buffer
	if resolveColor("unknown", &buf) {
		t.Error("expected color=false for unknown mode + non-TTY writer")
	}
}

// ---- Print — raw format -----------------------------------------------------

func TestPrint_Raw_WritesBodyOnly(t *testing.T) {
	body := []byte(`{"id":1}`)
	resp := makeResp(200, "application/json", body)

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "raw"})

	if !bytes.Equal(out.Bytes(), body) {
		t.Errorf("body: got %q, want %q", out.Bytes(), body)
	}
	if diag.Len() != 0 {
		t.Errorf("diag: expected empty, got %q", diag.String())
	}
}

func TestPrint_Raw_EmptyBody(t *testing.T) {
	resp := makeResp(204, "", []byte{})
	resp.Status = "204 No Content"

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "raw"})

	if out.Len() != 0 {
		t.Errorf("expected empty stdout for empty body, got %q", out.String())
	}
}

// ---- Print — pretty format --------------------------------------------------

func TestPrint_Pretty_StatusLineOnDiag(t *testing.T) {
	resp := makeResp(200, "text/plain", []byte("hello\n"))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never"})

	if !strings.Contains(diag.String(), "HTTP/1.1") {
		t.Errorf("status line missing from diag: %q", diag.String())
	}
	if !strings.Contains(diag.String(), "200 OK") {
		t.Errorf("status code missing from diag: %q", diag.String())
	}
}

func TestPrint_Pretty_BodyOnStdout(t *testing.T) {
	resp := makeResp(200, "text/plain", []byte("hello\n"))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never"})

	if out.String() != "hello\n" {
		t.Errorf("stdout: got %q, want %q", out.String(), "hello\n")
	}
}

func TestPrint_Pretty_JSONPrettyPrinted(t *testing.T) {
	raw := []byte(`{"id":1,"title":"hello"}`)
	resp := makeResp(200, "application/json", raw)

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never"})

	got := out.String()
	if !strings.Contains(got, "\n") {
		t.Errorf("JSON body was not pretty-printed: %q", got)
	}
	if !strings.Contains(got, `"id": 1`) {
		t.Errorf("JSON field missing: %q", got)
	}
}

func TestPrint_Pretty_NoHeaders_ByDefault(t *testing.T) {
	resp := makeResp(200, "text/plain", []byte("ok\n"))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never", ShowHeaders: false})

	// Diag should have only the status line (no "Content-Type:" line).
	if strings.Contains(diag.String(), "Content-Type") {
		t.Errorf("headers unexpectedly present in diag: %q", diag.String())
	}
}

func TestPrint_Pretty_ShowHeaders(t *testing.T) {
	resp := makeResp(200, "application/json", []byte(`{}`))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never", ShowHeaders: true})

	if !strings.Contains(diag.String(), "Content-Type") {
		t.Errorf("headers missing from diag: %q", diag.String())
	}
	// A blank line should appear between headers and body section.
	if !strings.Contains(diag.String(), "\n\n") {
		t.Errorf("blank line separator missing from diag: %q", diag.String())
	}
}

// ---- Quiet mode -------------------------------------------------------------

func TestPrint_Quiet_SuppressesStatusLine(t *testing.T) {
	resp := makeResp(200, "text/plain", []byte("hello\n"))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never", Quiet: true})

	if diag.Len() != 0 {
		t.Errorf("quiet mode: diag should be empty, got %q", diag.String())
	}
	if out.String() != "hello\n" {
		t.Errorf("quiet mode: body: got %q, want \"hello\\n\"", out.String())
	}
}

func TestPrint_Quiet_SuppressesHeaders(t *testing.T) {
	resp := makeResp(200, "application/json", []byte(`{}`))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never", Quiet: true, ShowHeaders: true})

	// Even with ShowHeaders=true, quiet mode must suppress headers on diag.
	if strings.Contains(diag.String(), "Content-Type") {
		t.Errorf("quiet mode: headers present on diag despite Quiet=true: %q", diag.String())
	}
}

func TestPrint_Quiet_BodyStillFormatted(t *testing.T) {
	body := []byte(`{"id":1}`)
	resp := makeResp(200, "application/json", body)

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "pretty", Color: "never", Quiet: true})

	// Body should be pretty-printed (JSON).
	if !strings.Contains(out.String(), "\n") {
		t.Errorf("quiet mode: JSON body should still be pretty-printed: %q", out.String())
	}
}

// ---- JSON format ------------------------------------------------------------

func TestPrint_JSON_OutputsToBody(t *testing.T) {
	resp := makeResp(200, "application/json", []byte(`{"id":1}`))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "json"})

	if out.Len() == 0 {
		t.Error("json mode: expected output on stdout")
	}
	if diag.Len() != 0 {
		t.Errorf("json mode: diag should be empty, got %q", diag.String())
	}
}

func TestPrint_JSON_ContainsStatusCode(t *testing.T) {
	resp := makeResp(201, "application/json", []byte(`{}`))
	resp.Status = "201 Created"

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "json"})

	got := out.String()
	if !strings.Contains(got, `"status_code": 201`) {
		t.Errorf("json mode: status_code missing: %q", got)
	}
}

func TestPrint_JSON_ContainsBody(t *testing.T) {
	resp := makeResp(200, "text/plain", []byte("hello world"))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "json"})

	got := out.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("json mode: body missing from output: %q", got)
	}
}

func TestPrint_JSON_ContainsHeaders(t *testing.T) {
	resp := makeResp(200, "application/json", []byte(`{}`))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "json"})

	got := out.String()
	if !strings.Contains(got, "Content-Type") {
		t.Errorf("json mode: Content-Type header missing: %q", got)
	}
}

func TestPrint_JSON_ContainsDurationMS(t *testing.T) {
	resp := makeResp(200, "text/plain", []byte("ok"))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "json"})

	got := out.String()
	if !strings.Contains(got, "duration_ms") {
		t.Errorf("json mode: duration_ms missing: %q", got)
	}
}

func TestPrint_JSON_IsValidJSON(t *testing.T) {
	resp := makeResp(200, "application/json", []byte(`{"id":1}`))

	var out, diag bytes.Buffer
	Print(&out, &diag, resp, Options{Format: "json"})

	// Verify the output is valid JSON by attempting to decode it.
	var parsed map[string]interface{}
	if err := json.NewDecoder(&out).Decode(&parsed); err != nil {
		t.Errorf("json mode: output is not valid JSON: %v", err)
	}
}

// ---- DefaultOptions ---------------------------------------------------------

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Format != "pretty" {
		t.Errorf("Format: got %q, want \"pretty\"", opts.Format)
	}
	if opts.Color != "auto" {
		t.Errorf("Color: got %q, want \"auto\"", opts.Color)
	}
	if opts.ShowHeaders {
		t.Error("ShowHeaders: expected false")
	}
	if opts.Quiet {
		t.Error("Quiet: expected false")
	}
}
