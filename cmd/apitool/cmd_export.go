package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MendezCarl/sailor.git/internal/collection"
	"github.com/MendezCarl/sailor.git/internal/curl"
	"github.com/MendezCarl/sailor.git/internal/request"
)

// runExport dispatches `sailor export <format> ...`.
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

// runExportCurl implements `sailor export curl -f <file>` and
// `sailor export curl --collection <file> "Name"`.
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
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
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
