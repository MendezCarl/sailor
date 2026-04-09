package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/MendezCarl/sailor.git/internal/config"
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
	case "collection":
		os.Exit(runCollection(os.Args[2:], cwd))
	case "env":
		os.Exit(runEnv(os.Args[2:], cwd))
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
	fmt.Fprintln(os.Stdout, "  send              Send an HTTP request from a YAML file")
	fmt.Fprintln(os.Stdout, "  run               Run a named request from a collection")
	fmt.Fprintln(os.Stdout, "  collection list   List collections in the current project")
	fmt.Fprintln(os.Stdout, "  collection show   Show details of a collection")
	fmt.Fprintln(os.Stdout, "  env list          List available environments")
	fmt.Fprintln(os.Stdout, "  env show          Show variables for an environment")
	fmt.Fprintln(os.Stdout, "  import curl       Import a curl command as a request YAML")
	fmt.Fprintln(os.Stdout, "  export curl       Export a request as a curl command")
	fmt.Fprintln(os.Stdout, "  version           Print version and exit")
	fmt.Fprintln(os.Stdout, "  help              Show this message")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Run 'sailor <command> --help' for command-specific usage.")
}
