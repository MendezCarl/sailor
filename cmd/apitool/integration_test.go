package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MendezCarl/sailor.git/internal/config"
)

// captureOutput redirects os.Stdout and os.Stderr, runs fn, then restores them.
// Returns whatever fn wrote to each stream.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origOut := os.Stdout
	origErr := os.Stderr
	defer func() {
		os.Stdout = origOut
		os.Stderr = origErr
	}()

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe (stdout): %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe (stderr): %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)
	rOut.Close()
	rErr.Close()

	return string(outBytes), string(errBytes)
}

// writeTmp writes content to a file in dir and returns the full path.
func writeTmp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTmp: %v", err)
	}
	return path
}

// ---- send from file ---------------------------------------------------------

func TestIntegration_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id":1,"name":"alice"}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s/users/1\n", srv.URL))

	cfg := config.Defaults()
	var code int
	stdout, _ := captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile}, cfg, t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, "alice") {
		t.Errorf("stdout: want body content, got %q", stdout)
	}
}

func TestIntegration_Send_RawOutput(t *testing.T) {
	const body = `{"raw":true}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s\n", srv.URL))

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--raw"}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if stdout != body {
		t.Errorf("stdout: got %q, want %q", stdout, body)
	}
	if stderr != "" {
		t.Errorf("stderr: want empty for --raw, got %q", stderr)
	}
}

func TestIntegration_Send_JSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":42}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s\n", srv.URL))

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--json"}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if stderr != "" {
		t.Errorf("stderr: want empty for --json, got %q", stderr)
	}

	var parsed map[string]interface{}
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&parsed); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\ngot: %s", err, stdout)
	}
	if _, ok := parsed["status_code"]; !ok {
		t.Errorf("json output: missing status_code field")
	}
}

func TestIntegration_Send_QuietMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s\n", srv.URL))

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--quiet"}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if stderr != "" {
		t.Errorf("quiet mode: stderr should be empty, got %q", stderr)
	}
	if strings.TrimRight(stdout, "\n") != "hello" {
		t.Errorf("quiet mode: stdout: got %q, want %q", stdout, "hello")
	}
}

func TestIntegration_Send_FailOnError_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s\n", srv.URL))

	var code int
	captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--fail-on-error", "--quiet"}, config.Defaults(), t.TempDir())
	})

	if code != exitHTTPError {
		t.Errorf("exit code: got %d, want %d (exitHTTPError)", code, exitHTTPError)
	}
}

func TestIntegration_Send_No4xxExitWithoutFlag(t *testing.T) {
	// Without --fail-on-error, a 404 response still exits 0.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s\n", srv.URL))

	var code int
	captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--quiet"}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d (exitOK — no --fail-on-error)", code, exitOK)
	}
}

func TestIntegration_Send_MissingFile(t *testing.T) {
	var code int
	captureOutput(t, func() {
		code = runSend([]string{"-f", "/no/such/file.yaml"}, config.Defaults(), t.TempDir())
	})

	if code != exitToolError {
		t.Errorf("exit code: got %d, want %d (exitToolError)", code, exitToolError)
	}
}

func TestIntegration_Send_NetworkError(t *testing.T) {
	// Start a server, record its URL, then close it so nothing is listening.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf("method: GET\nurl: %s\n", url))

	var code int
	captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--timeout", "2s"}, config.Defaults(), t.TempDir())
	})

	if code != exitNetworkError {
		t.Errorf("exit code: got %d, want %d (exitNetworkError)", code, exitNetworkError)
	}
}

// ---- variable substitution --------------------------------------------------

func TestIntegration_Send_VarFlagSubstitution(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf(
		"method: GET\nurl: %s/${api_version}/items\n", srv.URL))

	var code int
	captureOutput(t, func() {
		code = runSend([]string{
			"-f", reqFile,
			"--raw",
			"--var", "api_version=v2",
		}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if receivedPath != "/v2/items" {
		t.Errorf("server received path %q, want %q", receivedPath, "/v2/items")
	}
}

func TestIntegration_Send_DotEnvSubstitution(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	dir := t.TempDir()
	// Use a deliberately unusual key name to avoid OS-env interference.
	writeTmp(t, dir, ".env", "APITOOL_TEST_APIPATH=v3/widgets\n")
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf(
		"method: GET\nurl: %s/${apitool_test_apipath}\n", srv.URL))

	var code int
	captureOutput(t, func() {
		code = runSend([]string{"-f", reqFile, "--raw"}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if receivedPath != "/v3/widgets" {
		t.Errorf("server received path %q, want %q", receivedPath, "/v3/widgets")
	}
}

func TestIntegration_Send_VarFlagOverridesDotEnv(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	dir := t.TempDir()
	writeTmp(t, dir, ".env", "APITOOL_TEST_VER=env-value\n")
	reqFile := writeTmp(t, dir, "req.yaml", fmt.Sprintf(
		"method: GET\nurl: %s/${apitool_test_ver}/items\n", srv.URL))

	var code int
	captureOutput(t, func() {
		code = runSend([]string{
			"-f", reqFile,
			"--raw",
			"--var", "apitool_test_ver=cli-value",
		}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if receivedPath != "/cli-value/items" {
		t.Errorf("server received path %q, want %q (--var should override .env)", receivedPath, "/cli-value/items")
	}
}

// ---- run from collection ----------------------------------------------------

func TestIntegration_Run_ByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":1}]`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	colFile := writeTmp(t, dir, "col.yaml", fmt.Sprintf(`
name: Test Collection
base_url: %s
requests:
  - name: List Items
    method: GET
    url: "${base_url}/items"
`, srv.URL))

	var code int
	stdout, _ := captureOutput(t, func() {
		code = runRun([]string{"List Items", "--collection", colFile, "--raw"}, config.Defaults(), t.TempDir())
	})

	if code != exitOK {
		t.Errorf("exit code: got %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout, `"id":1`) {
		t.Errorf("stdout: want body content, got %q", stdout)
	}
}

func TestIntegration_Run_RequestNotFound(t *testing.T) {
	dir := t.TempDir()
	colFile := writeTmp(t, dir, "col.yaml", `
name: Test Collection
requests:
  - name: Existing Request
    method: GET
    url: http://example.com
`)

	var code int
	_, stderr := captureOutput(t, func() {
		code = runRun([]string{"No Such Request", "--collection", colFile}, config.Defaults(), t.TempDir())
	})

	if code != exitToolError {
		t.Errorf("exit code: got %d, want %d (exitToolError)", code, exitToolError)
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr: want 'not found' message, got %q", stderr)
	}
}

func TestIntegration_Run_MissingCollectionFile(t *testing.T) {
	var code int
	captureOutput(t, func() {
		code = runRun([]string{"Any Request", "--collection", "/no/such/collection.yaml"}, config.Defaults(), t.TempDir())
	})

	if code != exitToolError {
		t.Errorf("exit code: got %d, want %d (exitToolError)", code, exitToolError)
	}
}

// ---- curl import / export ---------------------------------------------------

func TestIntegration_ImportCurl_SimpleGET(t *testing.T) {
	var code int
	stdout, _ := captureOutput(t, func() {
		code = runImportCurl([]string{`curl https://api.example.com/users`})
	})

	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}
	if !strings.Contains(stdout, "method:") {
		t.Errorf("stdout: missing method field, got %q", stdout)
	}
	if !strings.Contains(stdout, "url:") {
		t.Errorf("stdout: missing url field, got %q", stdout)
	}
	if !strings.Contains(stdout, "api.example.com") {
		t.Errorf("stdout: missing URL value, got %q", stdout)
	}
}

func TestIntegration_ImportCurl_POST_WithHeader(t *testing.T) {
	var code int
	stdout, _ := captureOutput(t, func() {
		code = runImportCurl([]string{
			`curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"alice"}'`,
		})
	})

	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}
	if !strings.Contains(stdout, "POST") {
		t.Errorf("stdout: missing POST method, got %q", stdout)
	}
	if !strings.Contains(stdout, "Content-Type") {
		t.Errorf("stdout: missing header, got %q", stdout)
	}
	if !strings.Contains(stdout, "alice") {
		t.Errorf("stdout: missing body content, got %q", stdout)
	}
}

func TestIntegration_ImportCurl_OutputToFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.yaml")

	var code int
	captureOutput(t, func() {
		code = runImportCurl([]string{
			`curl https://api.example.com/test`,
			"--output", outFile,
		})
	})

	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(data), "api.example.com") {
		t.Errorf("output file content: missing URL, got %q", string(data))
	}
}

func TestIntegration_ExportCurl_FromFile(t *testing.T) {
	dir := t.TempDir()
	reqFile := writeTmp(t, dir, "req.yaml", `
method: POST
url: https://api.example.com/users
headers:
  Content-Type: application/json
body: '{"name":"alice"}'
`)

	var code int
	stdout, _ := captureOutput(t, func() {
		code = runExportCurl([]string{"-f", reqFile})
	})

	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}
	if !strings.Contains(stdout, "curl") {
		t.Errorf("stdout: missing curl keyword, got %q", stdout)
	}
	if !strings.Contains(stdout, "api.example.com") {
		t.Errorf("stdout: missing URL, got %q", stdout)
	}
	if !strings.Contains(stdout, "Content-Type") {
		t.Errorf("stdout: missing header, got %q", stdout)
	}
}

func TestIntegration_ExportCurl_FromCollection(t *testing.T) {
	dir := t.TempDir()
	colFile := writeTmp(t, dir, "col.yaml", `
name: Test Collection
requests:
  - name: Get User
    method: GET
    url: https://api.example.com/users/1
    headers:
      Accept: application/json
`)

	var code int
	stdout, _ := captureOutput(t, func() {
		code = runExportCurl([]string{"--collection", colFile, "Get User"})
	})

	if code != 0 {
		t.Errorf("exit code: got %d, want 0", code)
	}
	if !strings.Contains(stdout, "api.example.com") {
		t.Errorf("stdout: missing URL, got %q", stdout)
	}
	if !strings.Contains(stdout, "Accept") {
		t.Errorf("stdout: missing header, got %q", stdout)
	}
}
