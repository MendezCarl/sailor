package main

import (
	"testing"
	"time"

	"github.com/MendezCarl/sailor.git/internal/config"
	"github.com/MendezCarl/sailor.git/internal/request"
)

// ---- resolveTimeout ---------------------------------------------------------

func TestResolveTimeout_CLIFlagWins(t *testing.T) {
	// CLI "60s" should win over request YAML "10s" and config default (30s).
	req := &request.Request{Timeout: "10s"}
	cfg := config.Defaults()

	got, err := resolveTimeout("60s", req, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 60*time.Second {
		t.Errorf("got %v, want 60s", got)
	}
}

func TestResolveTimeout_RequestYAMLOverridesConfig(t *testing.T) {
	// Per-request YAML "90s" should override the config default (30s).
	req := &request.Request{Timeout: "90s"}
	cfg := config.Defaults()

	got, err := resolveTimeout("", req, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 90*time.Second {
		t.Errorf("got %v, want 90s", got)
	}
}

func TestResolveTimeout_ConfigFallback(t *testing.T) {
	// No CLI flag, no per-request YAML: fall back to config default (30s).
	req := &request.Request{}
	cfg := config.Defaults()

	got, err := resolveTimeout("", req, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 30*time.Second {
		t.Errorf("got %v, want 30s (config.Defaults() default)", got)
	}
}

func TestResolveTimeout_ZeroDisablesTimeout(t *testing.T) {
	// "0" parses as zero duration, which executor treats as no timeout.
	req := &request.Request{}
	cfg := config.Defaults()

	got, err := resolveTimeout("0", req, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("got %v, want 0 (zero disables timeout)", got)
	}
}

func TestResolveTimeout_InvalidCLIValue(t *testing.T) {
	_, err := resolveTimeout("notaduration", &request.Request{}, config.Defaults())
	if err == nil {
		t.Error("expected error for invalid CLI --timeout value")
	}
}

func TestResolveTimeout_InvalidRequestYAMLValue(t *testing.T) {
	req := &request.Request{Timeout: "badvalue"}
	_, err := resolveTimeout("", req, config.Defaults())
	if err == nil {
		t.Error("expected error for invalid timeout in request YAML")
	}
}

func TestResolveTimeout_MinuteShorthand(t *testing.T) {
	got, err := resolveTimeout("2m", &request.Request{}, config.Defaults())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 2*time.Minute {
		t.Errorf("got %v, want 2m", got)
	}
}

// ---- buildOpts --------------------------------------------------------------

func TestBuildOpts_DefaultsFromConfig(t *testing.T) {
	cfg := config.Defaults()
	opts := buildOpts(cfg, false, false, false, false)
	if opts.Format != "pretty" {
		t.Errorf("Format: got %q, want \"pretty\"", opts.Format)
	}
	if opts.Color != "auto" {
		t.Errorf("Color: got %q, want \"auto\"", opts.Color)
	}
	if opts.Quiet {
		t.Error("Quiet should be false by default")
	}
}

func TestBuildOpts_RawOverridesJSON(t *testing.T) {
	// --raw should win over --json when both are set.
	opts := buildOpts(config.Defaults(), true, false, true, false)
	if opts.Format != "raw" {
		t.Errorf("Format: got %q, want \"raw\"", opts.Format)
	}
}

func TestBuildOpts_JSONFormat(t *testing.T) {
	opts := buildOpts(config.Defaults(), false, false, true, false)
	if opts.Format != "json" {
		t.Errorf("Format: got %q, want \"json\"", opts.Format)
	}
}

func TestBuildOpts_QuietFlag(t *testing.T) {
	opts := buildOpts(config.Defaults(), false, true, false, false)
	if !opts.Quiet {
		t.Error("Quiet should be true when quiet=true is passed")
	}
}

func TestBuildOpts_ShowHeaders(t *testing.T) {
	opts := buildOpts(config.Defaults(), false, false, false, true)
	if !opts.ShowHeaders {
		t.Error("ShowHeaders should be true when showHeaders=true is passed")
	}
}
