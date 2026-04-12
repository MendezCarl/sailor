package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MendezCarl/sailor.git/internal/collection"
	"github.com/MendezCarl/sailor.git/internal/config"
	"github.com/MendezCarl/sailor.git/internal/env"
	"github.com/MendezCarl/sailor.git/internal/executor"
	"github.com/MendezCarl/sailor.git/internal/render"
	"github.com/MendezCarl/sailor.git/internal/request"
)

// runRun implements `sailor run <name>`.
func runRun(args []string, cfg *config.Config, cwd string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		collectionFile    string
		timeout           string
		raw               bool
		quiet             bool
		jsonOutput        bool
		showHeaders       bool
		failOnError       bool
		envName           string
		colorFlag         string
		noPager           bool
		followRedirects   bool
		noFollowRedirects bool
		insecure          bool
		vars              varFlags
	)

	fs.StringVar(&collectionFile, "collection", "", "path to a collection YAML file")
	fs.StringVar(&timeout, "timeout", "", "request timeout, e.g. \"30s\", \"1m\" (overrides config; 0 disables timeout)")
	fs.BoolVar(&raw, "raw", false, "output response body only, verbatim (pipe-friendly)")
	fs.BoolVar(&quiet, "quiet", false, "suppress status line and metadata on stderr")
	fs.BoolVar(&quiet, "q", false, "suppress status line (shorthand for --quiet)")
	fs.BoolVar(&jsonOutput, "json", false, "output full response as JSON (status, headers, body)")
	fs.BoolVar(&showHeaders, "headers", false, "print response headers")
	fs.BoolVar(&showHeaders, "i", false, "print response headers (shorthand for --headers)")
	fs.BoolVar(&failOnError, "fail-on-error", false, "exit 3 on HTTP 4xx/5xx response")
	fs.StringVar(&envName, "env", "", "environment name to activate (e.g. staging, production)")
	fs.StringVar(&colorFlag, "color", "", "color output: auto, always, or never (overrides config)")
	fs.BoolVar(&noPager, "no-pager", false, "disable automatic paging through $PAGER")
	fs.BoolVar(&followRedirects, "follow-redirects", false, "follow HTTP redirects (overrides config)")
	fs.BoolVar(&noFollowRedirects, "no-follow-redirects", false, "do not follow HTTP redirects")
	fs.BoolVar(&insecure, "insecure", false, "skip TLS certificate verification (for self-signed certs)")
	fs.Var(&vars, "var", "set a variable: key=value (repeatable)")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor run <request-name> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --collection examples/posts-collection.yaml")
		fmt.Fprintln(os.Stderr, "  sailor run \"Get Post\" --collection examples/posts-collection.yaml --headers")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --timeout 60s")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --env staging")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --var base_url=http://localhost:8080")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --raw | jq '.[0].title'")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --json | jq '.status_code'")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --fail-on-error && echo ok")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --color never")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --no-pager")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --no-follow-redirects")
		fmt.Fprintln(os.Stderr, "  sailor run \"List Posts\" --insecure")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Without --collection, searches .apitool/collections/*.yaml in the current directory.")
	}

	// Go's flag package stops at the first non-flag argument, so the request
	// name must be separated before flag parsing if it appears first.
	// This allows both orderings:
	//   sailor run "Name" --collection file.yaml
	//   sailor run --collection file.yaml "Name"
	var requestName string
	var flagArgs []string

	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		requestName = args[0]
		flagArgs = args[1:]
	} else {
		flagArgs = args
	}

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	// If the name was not the first arg, it may appear after the flags.
	if requestName == "" && fs.NArg() > 0 {
		requestName = fs.Arg(0)
	}

	if requestName == "" {
		fmt.Fprintln(os.Stderr, "error: request name is required")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return 1
	}

	cliVars, err := parseVarFlags(vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	opts := buildOpts(cfg, raw, quiet, jsonOutput, showHeaders, noPager, colorFlag)

	// Resolve the collection file: CLI flag > config default_collection > directory search.
	if collectionFile == "" {
		collectionFile = cfg.DefaultCollection
	}

	var (
		col    *collection.Collection
		req    *request.Request
		envDir string
	)

	if collectionFile != "" {
		// Explicit collection file (from --collection or config).
		col, err = collection.LoadFile(collectionFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		req, err = collection.FindRequest(col, requestName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		envDir = filepath.Dir(collectionFile)
	} else {
		// Search the default collection directory in CWD.
		colDir := collection.DefaultCollectionDir(cwd)

		col, req, err = collection.Search(colDir, requestName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		envDir = cwd
	}

	// Inject the collection's base_url as the lowest-priority base variable.
	baseVars := env.Vars{}
	if col.BaseURL != "" {
		baseVars["base_url"] = col.BaseURL
	}

	if envName == "" {
		envName = cfg.DefaultEnv
	}
	envFilePath, resolvedEnvName := env.ResolveEnvFile(cwd, envName)

	return runRequestPipeline(req, pipelineOpts{
		EnvDir:      envDir,
		CLIVars:     cliVars,
		BaseVars:    baseVars,
		RenderOpts:  opts,
		Config:      cfg,
		CLITimeout:  timeout,
		FailOnError: failOnError,
		EnvFilePath: envFilePath,
		EnvName:     resolvedEnvName,
		ExecOpts: executor.Options{
			FollowRedirects: resolveFollowRedirects(followRedirects, noFollowRedirects, cfg),
			Insecure:        insecure,
		},
	})
}

// buildOpts constructs render.Options from the merged config and CLI flags.
// CLI flags always override config values. --raw overrides --json.
func buildOpts(cfg *config.Config, raw, quiet, jsonOutput, showHeaders, noPager bool, color string) render.Options {
	opts := render.Options{
		Format:      cfg.Output.Format,
		Color:       cfg.Output.Color,
		ShowHeaders: cfg.Output.ShowHeaders,
	}
	if quiet {
		opts.Quiet = true
	}
	if jsonOutput {
		opts.Format = "json"
	}
	if raw {
		opts.Format = "raw" // raw wins over --json
	}
	if showHeaders {
		opts.ShowHeaders = true
	}
	if noPager {
		opts.NoPager = true
	}
	// --color flag overrides the config value when provided.
	if color != "" {
		opts.Color = color
	}
	return opts
}
