package main

import (
	"fmt"
	"os"

	"github.com/MendezCarl/sailor.git/internal/config"
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
