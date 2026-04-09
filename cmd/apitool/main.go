package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MendezCarl/sailor.git/internal/collection"
	"github.com/MendezCarl/sailor.git/internal/config"
	"github.com/MendezCarl/sailor.git/internal/curl"
	"github.com/MendezCarl/sailor.git/internal/env"
	"github.com/MendezCarl/sailor.git/internal/executor"
	"github.com/MendezCarl/sailor.git/internal/render"
	"github.com/MendezCarl/sailor.git/internal/request"
	"gopkg.in/yaml.v3"
)

// Version metadata. Set at build time with:
//
//	go build -ldflags "-X main.version=0.1.0 -X main.commit=abc1234 -X main.buildDate=2026-03-28"
//
// Zero values are used for local development builds.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// Exit codes match docs/architecture.md §6.2. Do not change values without
// updating that document.
const (
	exitOK           = 0 // request sent, response received (any HTTP status)
	exitToolError    = 1 // bad input, missing file, config error, flag error
	exitNetworkError = 2 // connection refused, DNS failure, timeout
	exitHTTPError    = 3 // HTTP 4xx/5xx — only when --fail-on-error is set
)

func main() {
	if len(os.Args) < 2 {
		printWelcome()
		os.Exit(0)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not determine working directory: %s\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(exitToolError)
	}

	switch os.Args[1] {
	case "send":
		os.Exit(runSend(os.Args[2:], cfg, cwd))
	case "run":
		os.Exit(runRun(os.Args[2:], cfg, cwd))
	case "import":
		os.Exit(runImport(os.Args[2:]))
	case "export":
		os.Exit(runExport(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Fprintf(os.Stdout, "sailor %s (commit: %s, built: %s)\n", version, commit, buildDate)
	case "help", "--help", "-h":
		printWelcome()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", os.Args[1])
		printWelcome()
		os.Exit(1)
	}
}

// varFlags accumulates repeated --var key=value flags.
type varFlags []string

func (v *varFlags) String() string { return "" }
func (v *varFlags) Set(s string) error {
	*v = append(*v, s)
	return nil
}

// parseVarFlags converts a varFlags slice into an env.Vars map.
// Returns an error if any entry is not in key=value format.
func parseVarFlags(flags varFlags) (env.Vars, error) {
	vars := env.Vars{}
	for _, f := range flags {
		k, v, err := env.ParseVarFlag(f)
		if err != nil {
			return nil, err
		}
		vars[k] = v
	}
	return vars, nil
}

// execRequest applies variables, executes the request, and renders output.
// envDir is the directory from which .env files are loaded.
// cliTimeout overrides per-request and config timeouts when non-empty.
// failOnError causes exit code 3 when the response status is 4xx or 5xx.
// envFilePath and envName identify an optional active environment YAML file.
// Returns an exit code.
func execRequest(req *request.Request, envDir string, cliVars env.Vars, baseVars env.Vars, opts render.Options, cfg *config.Config, cliTimeout string, failOnError bool, envFilePath string, envName string) int {
	vars, err := env.Collect(envDir, baseVars, cliVars, envFilePath, envName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	resolved := env.Apply(req, vars)

	timeout, err := resolveTimeout(cliTimeout, resolved, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	resp, err := executor.Send(resolved, timeout)
	if err != nil {
		var netErr *executor.NetworkError
		if errors.As(err, &netErr) {
			fmt.Fprintf(os.Stderr, "error: could not connect: %s\n", err)
			return exitNetworkError
		}
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	render.Print(os.Stdout, os.Stderr, resp, opts)

	if failOnError && resp.StatusCode >= 400 {
		return exitHTTPError
	}
	return exitOK
}

// resolveTimeout determines the effective timeout for a request.
// Priority: CLI flag > per-request YAML field > config value.
func resolveTimeout(cliTimeout string, req *request.Request, cfg *config.Config) (time.Duration, error) {
	if cliTimeout != "" {
		d, err := time.ParseDuration(cliTimeout)
		if err != nil {
			return 0, fmt.Errorf("invalid --timeout %q: use a duration like \"30s\" or \"1m\"", cliTimeout)
		}
		return d, nil
	}
	if req.Timeout != "" {
		d, err := time.ParseDuration(req.Timeout)
		if err != nil {
			return 0, fmt.Errorf("invalid timeout in request file %q: use a duration like \"30s\" or \"1m\"", req.Timeout)
		}
		return d, nil
	}
	return cfg.Timeout.Duration(), nil
}

// runSend implements `apitool send -f <file>`.
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

// runRun implements `apitool run <name>`.
func runRun(args []string, cfg *config.Config, cwd string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		collectionFile string
		timeout        string
		raw            bool
		quiet          bool
		jsonOutput     bool
		showHeaders    bool
		failOnError    bool
		envName        string
		vars           varFlags
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
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Without --collection, searches .apitool/collections/*.yaml in the current directory.")
	}

	// Go's flag package stops at the first non-flag argument, so the request
	// name must be separated before flag parsing if it appears first.
	// This allows both orderings:
	//   apitool run "Name" --collection file.yaml
	//   apitool run --collection file.yaml "Name"
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

	opts := buildOpts(cfg, raw, quiet, jsonOutput, showHeaders)

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

	return execRequest(req, envDir, cliVars, baseVars, opts, cfg, timeout, failOnError, envFilePath, resolvedEnvName)
}

// buildOpts constructs render.Options from the merged config and CLI flags.
// CLI flags always override config values. --raw overrides --json.
func buildOpts(cfg *config.Config, raw, quiet, jsonOutput, showHeaders bool) render.Options {
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
	return opts
}

// runImport dispatches `apitool import <format> ...`.
func runImport(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "Usage: sailor import <format> [args]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Formats:")
		fmt.Fprintln(os.Stderr, "  curl   Import a curl command")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Run 'sailor import <format> --help' for format-specific usage.")
		if len(args) == 0 {
			return 1
		}
		return 0
	}
	switch args[0] {
	case "curl":
		return runImportCurl(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown import format %q\n\n", args[0])
		fmt.Fprintln(os.Stderr, "Supported formats: curl")
		return 1
	}
}

// runImportCurl implements `apitool import curl "<curl command>"`.
func runImportCurl(args []string) int {
	fs := flag.NewFlagSet("import curl", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var outputFile string
	fs.StringVar(&outputFile, "output", "", "write YAML to this file instead of stdout")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor import curl \"<curl command>\" [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Parses a curl command and prints the equivalent request YAML.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  sailor import curl 'curl https://api.example.com/users'")
		fmt.Fprintln(os.Stderr, `  sailor import curl 'curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '"'"'{"name":"alice"}'"'"''`)
		fmt.Fprintln(os.Stderr, "  sailor import curl 'curl https://api.example.com/users' --output request.yaml")
	}

	// Extract positional curl string before flag parsing (it may contain
	// characters that the flag parser would misinterpret as flags).
	var curlStr string
	var flagArgs []string
	if len(args) > 0 && !hasPrefix(args[0], "-") {
		curlStr = args[0]
		flagArgs = args[1:]
	} else {
		flagArgs = args
	}

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if curlStr == "" && fs.NArg() > 0 {
		curlStr = fs.Arg(0)
	}
	if curlStr == "" {
		fmt.Fprintln(os.Stderr, "error: curl command string is required")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return 1
	}

	req, warnings, err := curl.Parse(curlStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	yamlBytes, err := marshalRequest(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not generate YAML: %s\n", err)
		return 1
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, yamlBytes, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: could not write %s: %s\n", outputFile, err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "written to %s\n", outputFile)
		return 0
	}

	os.Stdout.Write(yamlBytes) //nolint:errcheck
	return 0
}

// runExport dispatches `apitool export <format> ...`.
func runExport(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "Usage: sailor export <format> [args]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Formats:")
		fmt.Fprintln(os.Stderr, "  curl   Export a request as a curl command")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Run 'sailor export <format> --help' for format-specific usage.")
		if len(args) == 0 {
			return 1
		}
		return 0
	}
	switch args[0] {
	case "curl":
		return runExportCurl(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown export format %q\n\n", args[0])
		fmt.Fprintln(os.Stderr, "Supported formats: curl")
		return 1
	}
}

// runExportCurl implements `apitool export curl -f <file>` and
// `apitool export curl --collection <file> "Name"`.
func runExportCurl(args []string) int {
	fs := flag.NewFlagSet("export curl", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		filePath       string
		collectionFile string
	)

	fs.StringVar(&filePath, "f", "", "path to a request YAML file")
	fs.StringVar(&collectionFile, "collection", "", "path to a collection YAML file")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor export curl [flags] [request-name]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Converts a saved request to a curl command and prints it to stdout.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  sailor export curl -f examples/demo.yaml")
		fmt.Fprintln(os.Stderr, `  sailor export curl --collection examples/posts-collection.yaml "Get Post"`)
	}

	// Extract a positional request name before parsing flags.
	var requestName string
	var flagArgs []string
	if len(args) > 0 && !hasPrefix(args[0], "-") {
		requestName = args[0]
		flagArgs = args[1:]
	} else {
		flagArgs = args
	}

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if requestName == "" && fs.NArg() > 0 {
		requestName = fs.Arg(0)
	}

	var req *request.Request

	switch {
	case filePath != "":
		var err error
		req, err = request.LoadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}

	case collectionFile != "":
		if requestName == "" {
			fmt.Fprintln(os.Stderr, "error: request name is required when using --collection")
			fmt.Fprintln(os.Stderr)
			fs.Usage()
			return 1
		}
		col, err := collection.LoadFile(collectionFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		req, err = collection.FindRequest(col, requestName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}

	default:
		fmt.Fprintln(os.Stderr, "error: -f or --collection is required")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return 1
	}

	fmt.Fprintln(os.Stdout, curl.Export(req))
	return 0
}

// marshalRequest converts a Request to YAML bytes suitable for stdout or file output.
// Uses a local type with omitempty so empty fields are not written.
func marshalRequest(req *request.Request) ([]byte, error) {
	type yamlReq struct {
		Name    string            `yaml:"name,omitempty"`
		Method  string            `yaml:"method"`
		URL     string            `yaml:"url"`
		Headers map[string]string `yaml:"headers,omitempty"`
		Params  map[string]string `yaml:"params,omitempty"`
		Body    string            `yaml:"body,omitempty"`
	}
	out := yamlReq{
		Name:    req.Name,
		Method:  req.Method,
		URL:     req.URL,
		Headers: req.Headers,
		Params:  req.Params,
		Body:    req.Body,
	}
	return yaml.Marshal(out)
}

// hasPrefix returns true if s starts with prefix.
// Used instead of strings.HasPrefix to avoid an import for a one-liner.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func printWelcome() {
	fmt.Fprintln(os.Stdout, "Welcome to Sailor — a lightweight CLI API client.")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Usage: sailor <command> [flags]")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Commands:")
	fmt.Fprintln(os.Stdout, "  send         Send an HTTP request from a YAML file")
	fmt.Fprintln(os.Stdout, "  run          Run a named request from a collection")
	fmt.Fprintln(os.Stdout, "  import curl  Import a curl command as a request YAML")
	fmt.Fprintln(os.Stdout, "  export curl  Export a request as a curl command")
	fmt.Fprintln(os.Stdout, "  version      Print version and exit")
	fmt.Fprintln(os.Stdout, "  help         Show this message")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Run 'sailor <command> --help' for command-specific usage.")
}
