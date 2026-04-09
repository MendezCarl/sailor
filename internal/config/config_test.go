package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---- helpers ----------------------------------------------------------------

func writeTempConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTempConfig: %v", err)
	}
	return path
}

// ---- Defaults ---------------------------------------------------------------

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Timeout.Duration() != 30*time.Second {
		t.Errorf("default timeout: got %v, want 30s", cfg.Timeout.Duration())
	}
	if cfg.Output.Format != "pretty" {
		t.Errorf("default format: got %q, want \"pretty\"", cfg.Output.Format)
	}
	if cfg.Output.Color != "auto" {
		t.Errorf("default color: got %q, want \"auto\"", cfg.Output.Color)
	}
	if cfg.Output.ShowHeaders {
		t.Error("default show_headers: want false")
	}
	if cfg.DefaultCollection != "" {
		t.Errorf("default default_collection: got %q, want \"\"", cfg.DefaultCollection)
	}
}

// ---- LoadFile ---------------------------------------------------------------

func TestLoadFile_Missing(t *testing.T) {
	cfg, err := LoadFile("/no/such/file.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestLoadFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "bad.yaml", "timeout: [not a scalar")

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFile_InvalidDuration(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "cfg.yaml", "timeout: \"notaduration\"")

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid duration string")
	}
}

func TestLoadFile_ValidDurationString(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "cfg.yaml", "timeout: \"45s\"")

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timeout.Duration() != 45*time.Second {
		t.Errorf("got %v, want 45s", cfg.Timeout.Duration())
	}
}

func TestLoadFile_ValidDurationInt(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "cfg.yaml", "timeout: 60")

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timeout.Duration() != 60*time.Second {
		t.Errorf("got %v, want 60s", cfg.Timeout.Duration())
	}
}

// ---- validate ---------------------------------------------------------------

func TestValidate_InvalidFormat(t *testing.T) {
	cfg := Defaults()
	cfg.Output.Format = "xml"
	if err := validate(cfg); err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestValidate_InvalidColor(t *testing.T) {
	cfg := Defaults()
	cfg.Output.Color = "yes"
	if err := validate(cfg); err == nil {
		t.Error("expected error for invalid color")
	}
}

func TestValidate_ValidValues(t *testing.T) {
	for _, format := range []string{"pretty", "raw", "json"} {
		for _, color := range []string{"auto", "always", "never"} {
			cfg := Defaults()
			cfg.Output.Format = format
			cfg.Output.Color = color
			if err := validate(cfg); err != nil {
				t.Errorf("format=%q color=%q: unexpected error: %v", format, color, err)
			}
		}
	}
}

func TestValidate_JSONFormat(t *testing.T) {
	cfg := Defaults()
	cfg.Output.Format = "json"
	if err := validate(cfg); err != nil {
		t.Errorf("\"json\" should be a valid output format: %v", err)
	}
}

// ---- merge ------------------------------------------------------------------

func TestMerge_ProjectOverridesGlobal(t *testing.T) {
	base := Defaults()
	base.Timeout = Duration{d: 10 * time.Second}
	base.Output.Color = "always"
	base.DefaultCollection = "global.yaml"

	override := &Config{
		Timeout:           Duration{d: 5 * time.Second},
		DefaultCollection: "project.yaml",
		Output: OutputConfig{
			Format: "raw",
		},
	}

	got := merge(base, override)

	if got.Timeout.Duration() != 5*time.Second {
		t.Errorf("timeout: got %v, want 5s", got.Timeout.Duration())
	}
	if got.DefaultCollection != "project.yaml" {
		t.Errorf("default_collection: got %q, want \"project.yaml\"", got.DefaultCollection)
	}
	if got.Output.Format != "raw" {
		t.Errorf("format: got %q, want \"raw\"", got.Output.Format)
	}
	// Color was not overridden — base value should be preserved.
	if got.Output.Color != "always" {
		t.Errorf("color: got %q, want \"always\"", got.Output.Color)
	}
}

func TestMerge_ZeroOverrideKeepsBase(t *testing.T) {
	base := Defaults()
	override := &Config{} // all zero values

	got := merge(base, override)

	if got.Timeout.Duration() != base.Timeout.Duration() {
		t.Errorf("timeout changed unexpectedly: got %v", got.Timeout.Duration())
	}
	if got.Output.Format != base.Output.Format {
		t.Errorf("format changed unexpectedly: got %q", got.Output.Format)
	}
}

func TestMerge_ShowHeadersTrue(t *testing.T) {
	base := Defaults() // ShowHeaders = false
	override := &Config{Output: OutputConfig{ShowHeaders: true}}

	got := merge(base, override)

	if !got.Output.ShowHeaders {
		t.Error("show_headers: expected true after override")
	}
}

func TestMerge_BaseNotMutated(t *testing.T) {
	base := Defaults()
	override := &Config{Output: OutputConfig{Format: "raw"}}

	merge(base, override)

	if base.Output.Format != "pretty" {
		t.Error("merge mutated base config")
	}
}

// ---- loadFromPaths (integration) -------------------------------------------

func TestLoadFromPaths_MissingBothFiles(t *testing.T) {
	cfg, err := loadFromPaths("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return defaults unchanged.
	if cfg.Output.Format != "pretty" {
		t.Errorf("expected defaults, got format %q", cfg.Output.Format)
	}
}

func TestLoadFromPaths_GlobalOnly(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeTempConfig(t, dir, "global.yaml", `
output:
  color: always
  format: raw
`)

	cfg, err := loadFromPaths(globalPath, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Output.Color != "always" {
		t.Errorf("color: got %q, want \"always\"", cfg.Output.Color)
	}
	if cfg.Output.Format != "raw" {
		t.Errorf("format: got %q, want \"raw\"", cfg.Output.Format)
	}
}

func TestLoadFromPaths_ProjectOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeTempConfig(t, dir, "global.yaml", `
timeout: "10s"
output:
  color: always
`)
	projectPath := writeTempConfig(t, dir, "project.yaml", `
timeout: "5s"
output:
  color: never
`)

	cfg, err := loadFromPaths(globalPath, projectPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timeout.Duration() != 5*time.Second {
		t.Errorf("timeout: got %v, want 5s", cfg.Timeout.Duration())
	}
	if cfg.Output.Color != "never" {
		t.Errorf("color: got %q, want \"never\"", cfg.Output.Color)
	}
}

func TestLoadFromPaths_InvalidGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	globalPath := writeTempConfig(t, dir, "global.yaml", "output:\n  format: badformat\n")

	_, err := loadFromPaths(globalPath, "")
	if err == nil {
		t.Error("expected error for invalid global config format value")
	}
}

func TestLoadFromPaths_InvalidProjectConfig(t *testing.T) {
	dir := t.TempDir()
	projectPath := writeTempConfig(t, dir, "project.yaml", "output:\n  color: rainbow\n")

	_, err := loadFromPaths("", projectPath)
	if err == nil {
		t.Error("expected error for invalid project config color value")
	}
}

func TestLoadFile_SchemaVersionParsed(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "cfg.yaml", "schema_version: 1\noutput:\n  format: raw\n")

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", cfg.SchemaVersion)
	}
}

func TestLoadFile_DefaultEnvParsed(t *testing.T) {
	dir := t.TempDir()
	path := writeTempConfig(t, dir, "cfg.yaml", "default_env: staging\n")

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultEnv != "staging" {
		t.Errorf("DefaultEnv = %q, want \"staging\"", cfg.DefaultEnv)
	}
}

func TestMerge_DefaultEnvPropagated(t *testing.T) {
	base := Defaults()
	base.DefaultEnv = "development"
	override := &Config{DefaultEnv: "production"}

	got := merge(base, override)

	if got.DefaultEnv != "production" {
		t.Errorf("DefaultEnv = %q, want \"production\"", got.DefaultEnv)
	}
}

func TestMerge_DefaultEnvEmptyKeepsBase(t *testing.T) {
	base := Defaults()
	base.DefaultEnv = "development"
	override := &Config{} // DefaultEnv is ""

	got := merge(base, override)

	if got.DefaultEnv != "development" {
		t.Errorf("DefaultEnv = %q, want \"development\"", got.DefaultEnv)
	}
}
