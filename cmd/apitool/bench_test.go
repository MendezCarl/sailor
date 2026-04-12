// Startup time benchmarks for the sailor CLI.
//
// Startup time target: under 100ms for --version or --help on modern hardware
// (see docs/roadmap.md §7 and CLAUDE.md performance constraints).
//
// Run benchmarks:
//
//	go test -bench=. -benchmem ./cmd/apitool/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MendezCarl/sailor.git/internal/collection"
	"github.com/MendezCarl/sailor.git/internal/config"
	"github.com/MendezCarl/sailor.git/internal/env"
	"github.com/MendezCarl/sailor.git/internal/request"
)

// BenchmarkConfigLoad measures the cost of loading and merging config from disk.
// Uses a temp dir with no config file so it exercises the Defaults() + load path.
func BenchmarkConfigLoad(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.Load(dir)
		if err != nil {
			b.Fatalf("config.Load: %v", err)
		}
	}
}

// BenchmarkCollectionLoad measures YAML parse time for a small collection file.
func BenchmarkCollectionLoad(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.yaml")
	content := `name: Bench Collection
base_url: https://api.example.com
requests:
  - name: Get User
    method: GET
    url: ${base_url}/users/1
  - name: List Items
    method: GET
    url: ${base_url}/items
  - name: Create Item
    method: POST
    url: ${base_url}/items
    headers:
      Content-Type: application/json
    body: '{"name":"test"}'
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("write collection: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collection.LoadFile(path)
		if err != nil {
			b.Fatalf("collection.LoadFile: %v", err)
		}
	}
}

// BenchmarkRequestLoad measures YAML parse time for a single request file.
func BenchmarkRequestLoad(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "req.yaml")
	content := `method: POST
url: https://api.example.com/users
headers:
  Content-Type: application/json
  Accept: application/json
body: '{"name":"alice","email":"alice@example.com"}'
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatalf("write request: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := request.LoadFile(path)
		if err != nil {
			b.Fatalf("request.LoadFile: %v", err)
		}
	}
}

// BenchmarkEnvCollect measures full variable resolution with no .env files.
func BenchmarkEnvCollect(b *testing.B) {
	dir := b.TempDir()
	baseVars := env.Vars{"base_url": "https://api.example.com"}
	cliVars := env.Vars{"api_version": "v2"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Collect(dir, baseVars, cliVars, "", "")
		if err != nil {
			b.Fatalf("env.Collect: %v", err)
		}
	}
}

// BenchmarkInterpolate measures variable substitution on a typical URL.
func BenchmarkInterpolate(b *testing.B) {
	vars := env.Vars{
		"base_url":    "https://api.example.com",
		"api_version": "v2",
		"user_id":     "42",
	}
	url := "${base_url}/${api_version}/users/${user_id}/posts"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, _ := env.Interpolate(url, vars)
		if result == "" {
			b.Fatal("empty interpolation result")
		}
	}
}

// BenchmarkFullStartup measures the combined cost of the operations that
// dominate the hot path for a single-request send: config load, request load,
// and variable resolution. Does not make a network call.
func BenchmarkFullStartup(b *testing.B) {
	dir := b.TempDir()
	reqPath := filepath.Join(dir, "req.yaml")
	reqContent := fmt.Sprintf("method: GET\nurl: https://api.example.com/items\n")
	if err := os.WriteFile(reqPath, []byte(reqContent), 0o644); err != nil {
		b.Fatalf("write request: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := config.Load(dir)
		if err != nil {
			b.Fatalf("config.Load: %v", err)
		}
		req, err := request.LoadFile(reqPath)
		if err != nil {
			b.Fatalf("request.LoadFile: %v", err)
		}
		vars, err := env.Collect(dir, env.Vars{}, env.Vars{}, "", "")
		if err != nil {
			b.Fatalf("env.Collect: %v", err)
		}
		_, err = env.Apply(req, vars)
		if err != nil {
			b.Fatalf("env.Apply: %v", err)
		}
		_ = cfg
	}
}
