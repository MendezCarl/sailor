package curl

import (
	"strings"
	"testing"
)

// ---- tokenize ---------------------------------------------------------------

func TestTokenize_Simple(t *testing.T) {
	got := tokenize("curl https://example.com")
	want := []string{"curl", "https://example.com"}
	assertTokens(t, got, want)
}

func TestTokenize_SingleQuoted(t *testing.T) {
	got := tokenize("curl 'https://example.com/path?a=1&b=2'")
	want := []string{"curl", "https://example.com/path?a=1&b=2"}
	assertTokens(t, got, want)
}

func TestTokenize_DoubleQuoted(t *testing.T) {
	got := tokenize(`curl "https://example.com"`)
	want := []string{"curl", "https://example.com"}
	assertTokens(t, got, want)
}

func TestTokenize_DoubleQuotedEscape(t *testing.T) {
	got := tokenize(`curl -d "{\"key\":\"val\"}"`)
	want := []string{"curl", "-d", `{"key":"val"}`}
	assertTokens(t, got, want)
}

func TestTokenize_LineContinuation(t *testing.T) {
	input := "curl \\\n  -X POST \\\n  https://example.com"
	got := tokenize(input)
	want := []string{"curl", "-X", "POST", "https://example.com"}
	assertTokens(t, got, want)
}

func TestTokenize_EmptySingleQuote(t *testing.T) {
	got := tokenize("curl -d ''")
	want := []string{"curl", "-d", ""}
	assertTokens(t, got, want)
}

// ---- Parse — basic GET ------------------------------------------------------

func TestParse_SimpleGET(t *testing.T) {
	req, warns, err := Parse("curl https://api.example.com/users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if req.Method != "GET" {
		t.Errorf("method: got %q, want \"GET\"", req.Method)
	}
	if req.URL != "https://api.example.com/users" {
		t.Errorf("url: got %q", req.URL)
	}
	if req.Body != "" {
		t.Errorf("body: expected empty, got %q", req.Body)
	}
}

func TestParse_ExplicitMethod(t *testing.T) {
	req, _, err := Parse("curl -X DELETE https://api.example.com/users/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "DELETE" {
		t.Errorf("method: got %q, want \"DELETE\"", req.Method)
	}
}

func TestParse_LongMethodFlag(t *testing.T) {
	req, _, err := Parse("curl --request PUT https://api.example.com/users/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "PUT" {
		t.Errorf("method: got %q, want \"PUT\"", req.Method)
	}
}

// ---- Parse — POST with body -------------------------------------------------

func TestParse_PostWithBody(t *testing.T) {
	req, _, err := Parse(`curl -X POST https://api.example.com/users -d '{"name":"alice"}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("method: got %q, want \"POST\"", req.Method)
	}
	if req.Body != `{"name":"alice"}` {
		t.Errorf("body: got %q", req.Body)
	}
}

func TestParse_ImplicitPostFromBody(t *testing.T) {
	req, _, err := Parse(`curl https://api.example.com/users -d '{"name":"alice"}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("method: got %q, want \"POST\"", req.Method)
	}
}

func TestParse_DataRaw(t *testing.T) {
	req, _, err := Parse(`curl -X POST https://example.com --data-raw 'raw body'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Body != "raw body" {
		t.Errorf("body: got %q, want \"raw body\"", req.Body)
	}
}

// ---- Parse — headers --------------------------------------------------------

func TestParse_SingleHeader(t *testing.T) {
	req, _, err := Parse(`curl https://api.example.com -H "Accept: application/json"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header: got %q", req.Headers["Accept"])
	}
}

func TestParse_MultipleHeaders(t *testing.T) {
	input := `curl https://api.example.com ` +
		`-H "Accept: application/json" ` +
		`-H "Authorization: Bearer token123"`
	req, _, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers["Accept"] != "application/json" {
		t.Errorf("Accept: got %q", req.Headers["Accept"])
	}
	if req.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization: got %q", req.Headers["Authorization"])
	}
}

// ---- Parse — basic auth -----------------------------------------------------

func TestParse_BasicAuth(t *testing.T) {
	req, _, err := Parse(`curl https://api.example.com -u admin:secret`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should produce Authorization: Basic base64(admin:secret)
	auth := req.Headers["Authorization"]
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("Authorization header: got %q, want Basic prefix", auth)
	}
}

// ---- Parse — multiline ------------------------------------------------------

func TestParse_Multiline(t *testing.T) {
	input := "curl \\\n" +
		"  -X POST \\\n" +
		"  https://api.example.com \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"foo\":\"bar\"}'"
	req, warns, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if req.Method != "POST" {
		t.Errorf("method: got %q", req.Method)
	}
	if req.URL != "https://api.example.com" {
		t.Errorf("url: got %q", req.URL)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type: got %q", req.Headers["Content-Type"])
	}
	if req.Body != `{"foo":"bar"}` {
		t.Errorf("body: got %q", req.Body)
	}
}

// ---- Parse — query params ---------------------------------------------------

func TestParse_QueryParamsInURL(t *testing.T) {
	req, _, err := Parse("curl 'https://api.example.com/posts?page=2&limit=10'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL != "https://api.example.com/posts" {
		t.Errorf("url: got %q, want path without query string", req.URL)
	}
	if req.Params["page"] != "2" {
		t.Errorf("param page: got %q, want \"2\"", req.Params["page"])
	}
	if req.Params["limit"] != "10" {
		t.Errorf("param limit: got %q, want \"10\"", req.Params["limit"])
	}
}

// ---- Parse — unsupported flags ----------------------------------------------

func TestParse_UnsupportedFlagsWarn(t *testing.T) {
	_, warns, err := Parse("curl https://example.com --retry 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) == 0 {
		t.Error("expected a warning for --retry")
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "--retry") {
			found = true
		}
	}
	if !found {
		t.Errorf("warning did not mention --retry: %v", warns)
	}
}

func TestParse_SilentlySilent(t *testing.T) {
	// -s (--silent) and -L (--location) are recognized and silently ignored.
	_, warns, err := Parse("curl -s -L https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("expected no warnings for -s -L, got: %v", warns)
	}
}

// ---- Parse — --json flag ----------------------------------------------------

func TestParse_JsonFlag(t *testing.T) {
	req, _, err := Parse(`curl --json '{"name":"alice"}' https://api.example.com/users`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type: got %q", req.Headers["Content-Type"])
	}
	if req.Headers["Accept"] != "application/json" {
		t.Errorf("Accept: got %q", req.Headers["Accept"])
	}
	if req.Body != `{"name":"alice"}` {
		t.Errorf("body: got %q", req.Body)
	}
}

// ---- Parse — error cases ----------------------------------------------------

func TestParse_Empty(t *testing.T) {
	_, _, err := Parse("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParse_NotCurl(t *testing.T) {
	_, _, err := Parse("wget https://example.com")
	if err == nil {
		t.Error("expected error for non-curl command")
	}
}

func TestParse_NoURL(t *testing.T) {
	_, _, err := Parse("curl -X GET")
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

// ---- helpers ----------------------------------------------------------------

func assertTokens(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("token count: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}
