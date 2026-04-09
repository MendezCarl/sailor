package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MendezCarl/sailor.git/internal/config"
	"github.com/MendezCarl/sailor.git/internal/env"
	"github.com/MendezCarl/sailor.git/internal/request"
)

// runSend implements `sailor send -f <file>`.
func runSend(args []string, cfg *config.Config, cwd string) int {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		filePath    string
		timeout     string
		raw         bool
		quiet       bool
		jsonOutput  bool
		showHeaders bool
		failOnError bool
		envName     string
		vars        varFlags
	)

	fs.StringVar(&filePath, "f", "", "path to a request YAML file (required)")
	fs.StringVar(&timeout, "timeout", "", "request timeout, e.g. \"30s\", \"1m\" (overrides config; 0 disables timeout)")
	fs.BoolVar(&raw, "raw", false, "output response body only, verbatim (pipe-friendly)")
	fs.BoolVar(&quiet, "quiet", false, "suppress status line and metadata on stderr")
	fs.BoolVar(&quiet, "q", false, "suppress status line (shorthand for --quiet)")
	fs.BoolVar(&jsonOutput, "json", false, "output full response as JSON (status, headers, body)")
	fs.BoolVar(&showHeaders, "headers", false, "print response headers")
	fs.BoolVar(&showHeaders, "i", false, "print response headers (shorthand for --headers)")
	fs.BoolVar(&failOnError, "fail-on-error", false, "exit 3 on HTTP 4xx/5xx response")
	fs.StringVar(&envName, "env", "", "environment name to activate (e.g. staging, production)")
	fs.Var(&vars, "var", "set a variable: key=value (repeatable)")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor send -f <file> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --headers")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --timeout 60s")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --env staging")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --var base_url=http://localhost:8080")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --raw | jq '.title'")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --json | jq '.status_code'")
		fmt.Fprintln(os.Stderr, "  sailor send -f examples/demo.yaml --fail-on-error && echo ok")
	}

	if err := fs.Parse(args); err != nil {
		return exitToolError
	}
	if filePath == "" {
		fmt.Fprintln(os.Stderr, "error: -f flag is required")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return exitToolError
	}

	req, err := request.LoadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	cliVars, err := parseVarFlags(vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	if envName == "" {
		envName = cfg.DefaultEnv
	}
	envFilePath, resolvedEnvName := env.ResolveEnvFile(cwd, envName)

	opts := buildOpts(cfg, raw, quiet, jsonOutput, showHeaders)
	envDir := filepath.Dir(filePath)
	return execRequest(req, envDir, cliVars, env.Vars{}, opts, cfg, timeout, failOnError, envFilePath, resolvedEnvName)
}
