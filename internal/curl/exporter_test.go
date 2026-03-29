package curl

import (
	"strings"
	"testing"

	"github.com/MendezCarl/sailor.git/internal/request"
)

func TestExport_SimpleGET(t *testing.T) {
	req := &request.Request{
		Method: "GET",
		URL:    "https://api.example.com/users",
	}
	got := Export(req)
	if !strings.Contains(got, "https://api.example.com/users") {
		t.Errorf("URL missing: %q", got)
	}
	// GET should not include --request GET
	if strings.Contains(got, "--request") {
		t.Errorf("unexpected --request in GET output: %q", got)
	}
}

func TestExport_GETWithHeaders(t *testing.T) {
	req := &request.Request{
		Method: "GET",
		URL:    "https://api.example.com/users",
		Headers: map[string]string{
			"Accept":        "application/json",
			"Authorization": "Bearer tok",
		},
	}
	got := Export(req)
	if !strings.Contains(got, "--header") {
		t.Errorf("--header missing: %q", got)
	}
	if !strings.Contains(got, "Accept: application/json") {
		t.Errorf("Accept header missing: %q", got)
	}
	if !strings.Contains(got, "Authorization: Bearer tok") {
		t.Errorf("Authorization header missing: %q", got)
	}
}

func TestExport_HeadersAreSorted(t *testing.T) {
	req := &request.Request{
		Method: "GET",
		URL:    "https://api.example.com/users",
		Headers: map[string]string{
			"Z-Custom": "last",
			"Accept":   "application/json",
		},
	}
	got := Export(req)
	acceptPos := strings.Index(got, "Accept")
	customPos := strings.Index(got, "Z-Custom")
	if acceptPos > customPos {
		t.Errorf("headers not sorted: Accept should come before Z-Custom\n%s", got)
	}
}

func TestExport_PostWithBody(t *testing.T) {
	req := &request.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"name":"alice"}`,
	}
	got := Export(req)
	if !strings.Contains(got, "--request POST") {
		t.Errorf("--request POST missing: %q", got)
	}
	if !strings.Contains(got, "--data") {
		t.Errorf("--data missing: %q", got)
	}
	if !strings.Contains(got, `{"name":"alice"}`) {
		t.Errorf("body missing: %q", got)
	}
}

func TestExport_MultilineFormat(t *testing.T) {
	req := &request.Request{
		Method: "POST",
		URL:    "https://api.example.com",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"x":1}`,
	}
	got := Export(req)
	// Must use backslash continuation.
	if !strings.Contains(got, "\\\n") {
		t.Errorf("expected multi-line format with \\<newline>: %q", got)
	}
}

func TestExport_QueryParams(t *testing.T) {
	req := &request.Request{
		Method: "GET",
		URL:    "https://api.example.com/posts",
		Params: map[string]string{
			"page":  "2",
			"limit": "10",
		},
	}
	got := Export(req)
	if !strings.Contains(got, "page=2") {
		t.Errorf("query param page missing: %q", got)
	}
	if !strings.Contains(got, "limit=10") {
		t.Errorf("query param limit missing: %q", got)
	}
}

func TestExport_SingleQuotesValues(t *testing.T) {
	req := &request.Request{
		Method: "GET",
		URL:    "https://api.example.com/users",
	}
	got := Export(req)
	if !strings.Contains(got, "'https://") {
		t.Errorf("URL not single-quoted: %q", got)
	}
}

func TestExport_PreservesVariableReferences(t *testing.T) {
	req := &request.Request{
		Method: "GET",
		URL:    "${base_url}/users",
	}
	got := Export(req)
	if !strings.Contains(got, "${base_url}/users") {
		t.Errorf("variable reference not preserved: %q", got)
	}
}

func TestExport_SingleQuoteEscaping(t *testing.T) {
	// A header value with a single quote must be properly escaped.
	val := "it's a test"
	got := singleQuote(val)
	// Expected: 'it'"'"'s a test'
	want := "'it'\"'\"'s a test'"
	if got != want {
		t.Errorf("singleQuote(%q): got %q, want %q", val, got, want)
	}
}

// ---- Round-trip -------------------------------------------------------------

func TestRoundTrip_PostJSON(t *testing.T) {
	original := `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"alice"}'`

	req, _, err := Parse(original)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	exported := Export(req)

	// Re-parse the exported command to verify semantic equivalence.
	req2, _, err := Parse(exported)
	if err != nil {
		t.Fatalf("re-Parse error: %v", err)
	}

	if req.Method != req2.Method {
		t.Errorf("method: %q → %q", req.Method, req2.Method)
	}
	if req.URL != req2.URL {
		t.Errorf("url: %q → %q", req.URL, req2.URL)
	}
	if req.Body != req2.Body {
		t.Errorf("body: %q → %q", req.Body, req2.Body)
	}
}
