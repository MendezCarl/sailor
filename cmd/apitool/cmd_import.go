package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/curl"
)

// runImport dispatches `sailor import <format> ...`.
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

// runImportCurl implements `sailor import curl "<curl command>"`.
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
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
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

	yamlBytes, err := requestToYAML(req)
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
