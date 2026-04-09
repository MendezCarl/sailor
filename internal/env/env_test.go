package env

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Interpolate ---

func TestInterpolate_NoVars(t *testing.T) {
	got, undef := Interpolate("https://example.com/users", Vars{})
	if got != "https://example.com/users" {
		t.Errorf("got %q, want unchanged string", got)
	}
	if len(undef) != 0 {
		t.Errorf("expected no undefined vars, got %v", undef)
	}
}

func TestInterpolate_SingleVar(t *testing.T) {
	vars := Vars{"base_url": "https://api.example.com"}
	got, undef := Interpolate("${base_url}/users", vars)
	if got != "https://api.example.com/users" {
		t.Errorf("got %q, want %q", got, "https://api.example.com/users")
	}
	if len(undef) != 0 {
		t.Errorf("expected no undefined vars, got %v", undef)
	}
}

func TestInterpolate_MultipleVars(t *testing.T) {
	vars := Vars{
		"base_url": "https://api.example.com",
		"user_id":  "42",
	}
	got, _ := Interpolate("${base_url}/users/${user_id}", vars)
	want := "https://api.example.com/users/42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInterpolate_UndefinedVar(t *testing.T) {
	got, undef := Interpolate("${base_url}/users", Vars{})
	// Undefined var is preserved verbatim.
	if got != "${base_url}/users" {
		t.Errorf("got %q, want literal passthrough", got)
	}
	if len(undef) != 1 || undef[0] != "base_url" {
		t.Errorf("expected [base_url] undefined, got %v", undef)
	}
}

func TestInterpolate_CaseInsensitive(t *testing.T) {
	// Keys in vars are lowercase; references may be any case.
	vars := Vars{"auth_token": "secret"}
	got, undef := Interpolate("Bearer ${AUTH_TOKEN}", vars)
	if got != "Bearer secret" {
		t.Errorf("got %q, want %q", got, "Bearer secret")
	}
	if len(undef) != 0 {
		t.Errorf("expected no undefined vars, got %v", undef)
	}
}

func TestInterpolate_UnclosedBrace(t *testing.T) {
	// Unclosed ${ is written verbatim; no panic.
	got, _ := Interpolate("${base_url", Vars{"base_url": "x"})
	if got != "${base_url" {
		t.Errorf("got %q, want verbatim passthrough", got)
	}
}

func TestInterpolate_EmptyString(t *testing.T) {
	got, undef := Interpolate("", Vars{})
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if len(undef) != 0 {
		t.Errorf("expected no undefined, got %v", undef)
	}
}

func TestInterpolate_VarInMiddle(t *testing.T) {
	vars := Vars{"env": "staging"}
	got, _ := Interpolate("https://${env}.api.example.com/v1", vars)
	want := "https://staging.api.example.com/v1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- LoadDotEnv ---

func TestLoadDotEnv_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := `
# This is a comment
BASE_URL=https://api.example.com
AUTH_TOKEN=secret123
EMPTY_VAR=
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := LoadDotEnv(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Keys are normalised to lowercase.
	if vars["base_url"] != "https://api.example.com" {
		t.Errorf("base_url: got %q", vars["base_url"])
	}
	if vars["auth_token"] != "secret123" {
		t.Errorf("auth_token: got %q", vars["auth_token"])
	}
	if vars["empty_var"] != "" {
		t.Errorf("empty_var: got %q, want empty string", vars["empty_var"])
	}
}

func TestLoadDotEnv_QuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := `
DOUBLE="quoted value"
SINGLE='single quoted'
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := LoadDotEnv(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars["double"] != "quoted value" {
		t.Errorf("double: got %q, want %q", vars["double"], "quoted value")
	}
	if vars["single"] != "single quoted" {
		t.Errorf("single: got %q, want %q", vars["single"], "single quoted")
	}
}

func TestLoadDotEnv_MissingFile(t *testing.T) {
	vars, err := LoadDotEnv("/nonexistent/.env")
	if err != nil {
		t.Errorf("missing file should not error, got: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty vars, got %v", vars)
	}
}

func TestLoadDotEnv_SkipsInvalidLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	// Line without = is skipped tolerantly, not an error.
	content := "NOEQUALSSIGN\nVALID=yes\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := LoadDotEnv(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["valid"] != "yes" {
		t.Errorf("valid: got %q", vars["valid"])
	}
}

// --- Collect (priority ordering) ---

func TestCollect_CLIOverridesAll(t *testing.T) {
	dir := t.TempDir()
	// Write a .env file with a value.
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MY_VAR=from_dotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}

	base := Vars{"my_var": "from_base"}
	cli := Vars{"my_var": "from_cli"}

	vars, err := Collect(dir, base, cli, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// CLI value should win.
	if vars["my_var"] != "from_cli" {
		t.Errorf("my_var: got %q, want %q", vars["my_var"], "from_cli")
	}
}

func TestCollect_LocalOverridesDotEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MY_VAR=base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte("MY_VAR=local\n"), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := Collect(dir, Vars{}, Vars{}, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if vars["my_var"] != "local" {
		t.Errorf("my_var: got %q, want %q", vars["my_var"], "local")
	}
}

func TestCollect_BaseVarsAsLowestPriority(t *testing.T) {
	dir := t.TempDir()
	// No .env files, so only base vars and (potentially) OS env.
	// Use an unusual name unlikely to exist in the test environment.
	base := Vars{"apitool_test_unique_base_only": "base_value"}

	vars, err := Collect(dir, base, Vars{}, "", "")
	if err != nil {
		t.Fatal(err)
	}

	if vars["apitool_test_unique_base_only"] != "base_value" {
		t.Errorf("base var not found or wrong value: %q", vars["apitool_test_unique_base_only"])
	}
}

func TestCollect_EnvYAMLLayerPriority(t *testing.T) {
	dir := t.TempDir()
	// env YAML sets MY_VAR=from_yaml; .env overrides to from_dotenv
	envFile := filepath.Join(dir, "staging.yaml")
	if err := os.WriteFile(envFile, []byte("my_var: from_yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MY_VAR=from_dotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := Collect(dir, Vars{}, Vars{}, envFile, "")
	if err != nil {
		t.Fatal(err)
	}

	// .env wins over env YAML.
	if vars["my_var"] != "from_dotenv" {
		t.Errorf("my_var: got %q, want %q", vars["my_var"], "from_dotenv")
	}
}

func TestCollect_DotEnvEnvSpecific(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MY_VAR=base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.staging"), []byte("MY_VAR=staging\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte("MY_VAR=local\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// With staging env: .env.staging wins over .env but .env.local wins over both.
	vars, err := Collect(dir, Vars{}, Vars{}, "", "staging")
	if err != nil {
		t.Fatal(err)
	}
	if vars["my_var"] != "local" {
		t.Errorf("my_var: got %q, want \"local\"", vars["my_var"])
	}
}

func TestCollect_DotEnvEnvSpecificBetweenDotEnvAndLocal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("MY_VAR=base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.staging"), []byte("MY_VAR=staging\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// No .env.local — so .env.staging should win over .env.

	vars, err := Collect(dir, Vars{}, Vars{}, "", "staging")
	if err != nil {
		t.Fatal(err)
	}
	if vars["my_var"] != "staging" {
		t.Errorf("my_var: got %q, want \"staging\"", vars["my_var"])
	}
}

func TestCollect_NoEnvFilePath_Skipped(t *testing.T) {
	dir := t.TempDir()
	// No env YAML, no .env files — only base vars.
	base := Vars{"apitool_test_no_env_file": "base_only"}

	vars, err := Collect(dir, base, Vars{}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if vars["apitool_test_no_env_file"] != "base_only" {
		t.Errorf("got %q, want \"base_only\"", vars["apitool_test_no_env_file"])
	}
}

// --- LoadEnvFile ---

func writeEnvFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadEnvFile_SingleEnvFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "staging.yaml", "base_url: https://staging.example.com\nauth_token: tok123\n")

	vars, err := LoadEnvFile(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["base_url"] != "https://staging.example.com" {
		t.Errorf("base_url: got %q", vars["base_url"])
	}
	if vars["auth_token"] != "tok123" {
		t.Errorf("auth_token: got %q", vars["auth_token"])
	}
}

func TestLoadEnvFile_SingleEnvSkipsSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "env.yaml", "schema_version: 1\nbase_url: https://example.com\n")

	vars, err := LoadEnvFile(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := vars["schema_version"]; ok {
		t.Error("schema_version should not appear in returned vars")
	}
	if vars["base_url"] != "https://example.com" {
		t.Errorf("base_url: got %q", vars["base_url"])
	}
}

func TestLoadEnvFile_MultiEnvFormat_HappyPath(t *testing.T) {
	dir := t.TempDir()
	content := `environments:
  development:
    base_url: http://localhost:3000
  production:
    base_url: https://api.example.com
`
	path := writeEnvFile(t, dir, "environments.yaml", content)

	vars, err := LoadEnvFile(path, "development")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["base_url"] != "http://localhost:3000" {
		t.Errorf("base_url: got %q, want %q", vars["base_url"], "http://localhost:3000")
	}
}

func TestLoadEnvFile_MultiEnvFormat_UnknownEnv(t *testing.T) {
	dir := t.TempDir()
	content := "environments:\n  development:\n    base_url: http://localhost\n"
	path := writeEnvFile(t, dir, "environments.yaml", content)

	_, err := LoadEnvFile(path, "production")
	if err == nil {
		t.Fatal("expected error for unknown environment name")
	}
}

func TestLoadEnvFile_MultiEnvFormat_EmptyEnvName(t *testing.T) {
	dir := t.TempDir()
	content := "environments:\n  development:\n    base_url: http://localhost\n"
	path := writeEnvFile(t, dir, "environments.yaml", content)

	_, err := LoadEnvFile(path, "")
	if err == nil {
		t.Fatal("expected error when envName is empty for multi-env file")
	}
}

func TestLoadEnvFile_MissingFile(t *testing.T) {
	vars, err := LoadEnvFile("/no/such/file.yaml", "")
	if err != nil {
		t.Fatalf("missing file should not error, got: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty vars, got %v", vars)
	}
}

func TestLoadEnvFile_IntValuesCoercedToString(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "env.yaml", "port: 8080\nenabled: true\n")

	vars, err := LoadEnvFile(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["port"] != "8080" {
		t.Errorf("port: got %q, want \"8080\"", vars["port"])
	}
	if vars["enabled"] != "true" {
		t.Errorf("enabled: got %q, want \"true\"", vars["enabled"])
	}
}

// --- ResolveEnvFile ---

func TestResolveEnvFile_EmptyEnvName(t *testing.T) {
	fp, name := ResolveEnvFile(t.TempDir(), "")
	if fp != "" || name != "" {
		t.Errorf("expected empty results, got %q %q", fp, name)
	}
}

func TestResolveEnvFile_ProjectSingleEnv(t *testing.T) {
	dir := t.TempDir()
	envsDir := filepath.Join(dir, ".apitool", "envs")
	if err := os.MkdirAll(envsDir, 0755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(envsDir, "staging.yaml")
	if err := os.WriteFile(envPath, []byte("base_url: https://staging.example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fp, resolvedName := ResolveEnvFile(dir, "staging")
	if fp != envPath {
		t.Errorf("filePath: got %q, want %q", fp, envPath)
	}
	if resolvedName != "" {
		t.Errorf("resolvedEnvName: got %q, want \"\" (single-env file)", resolvedName)
	}
}

func TestResolveEnvFile_ProjectMultiEnv(t *testing.T) {
	dir := t.TempDir()
	envsDir := filepath.Join(dir, ".apitool", "envs")
	if err := os.MkdirAll(envsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// No staging.yaml — only environments.yaml (multi-env).
	multiPath := filepath.Join(envsDir, "environments.yaml")
	if err := os.WriteFile(multiPath, []byte("environments:\n  staging:\n    base_url: https://staging.example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	fp, resolvedName := ResolveEnvFile(dir, "staging")
	if fp != multiPath {
		t.Errorf("filePath: got %q, want %q", fp, multiPath)
	}
	if resolvedName != "staging" {
		t.Errorf("resolvedEnvName: got %q, want \"staging\"", resolvedName)
	}
}

func TestResolveEnvFile_NotFound(t *testing.T) {
	fp, name := ResolveEnvFile(t.TempDir(), "nonexistent")
	if fp != "" || name != "" {
		t.Errorf("expected empty results for not-found env, got %q %q", fp, name)
	}
}

// --- ParseVarFlag ---

func TestParseVarFlag_Valid(t *testing.T) {
	k, v, err := ParseVarFlag("base_url=https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "base_url" || v != "https://api.example.com" {
		t.Errorf("got k=%q v=%q", k, v)
	}
}

func TestParseVarFlag_ValueContainsEquals(t *testing.T) {
	// Only the first = is the separator; the rest is the value.
	k, v, err := ParseVarFlag("token=abc=def=xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "token" || v != "abc=def=xyz" {
		t.Errorf("got k=%q v=%q", k, v)
	}
}

func TestParseVarFlag_MissingEquals(t *testing.T) {
	_, _, err := ParseVarFlag("noequalssign")
	if err == nil {
		t.Error("expected error for missing =, got nil")
	}
}
