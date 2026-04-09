package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/MendezCarl/sailor.git/internal/env"
)

// runEnv dispatches `sailor env <subcommand>`.
func runEnv(args []string, cwd string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "Usage: sailor env <subcommand>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Subcommands:")
		fmt.Fprintln(os.Stderr, "  list   List available environments")
		fmt.Fprintln(os.Stderr, "  show   Show variables for a named environment")
		return 0
	}
	switch args[0] {
	case "list":
		return runEnvList(args[1:], cwd)
	case "show":
		return runEnvShow(args[1:], cwd)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown env subcommand %q\n\n", args[0])
		fmt.Fprintln(os.Stderr, "Subcommands: list, show")
		return exitToolError
	}
}

// runEnvList implements `sailor env list`.
func runEnvList(args []string, cwd string) int {
	fs := flag.NewFlagSet("env list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor env list [--json]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Lists all environments discoverable from the current project.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitToolError
	}

	entries, err := env.ListEnvs(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no environments found")
		fmt.Fprintln(os.Stderr, "hint: add YAML files to .apitool/envs/ to define environments")
		return exitToolError
	}

	if jsonOutput {
		type jsonEntry struct {
			Name   string `json:"name"`
			Source string `json:"source"`
			Multi  bool   `json:"multi"`
		}
		out := make([]jsonEntry, len(entries))
		for i, e := range entries {
			out[i] = jsonEntry{Name: e.Name, Source: e.Source, Multi: e.Multi}
		}
		data, err := json.Marshal(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		fmt.Fprintln(os.Stdout, string(data))
		return exitOK
	}

	const nameW = 20
	for _, e := range entries {
		suffix := ""
		if e.Multi {
			suffix = "  (multi)"
		}
		fmt.Fprintf(os.Stdout, "%-*s  %s%s\n", nameW, e.Name, e.Source, suffix)
	}
	return exitOK
}

// runEnvShow implements `sailor env show <name>`.
func runEnvShow(args []string, cwd string) int {
	fs := flag.NewFlagSet("env show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor env show <name> [--json]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Shows the variables defined in the named environment.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  sailor env show staging")
		fmt.Fprintln(os.Stderr, "  sailor env show production --json")
	}

	var nameArg string
	var flagArgs []string
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		nameArg = args[0]
		flagArgs = args[1:]
	} else {
		flagArgs = args
	}
	if err := fs.Parse(flagArgs); err != nil {
		return exitToolError
	}
	if nameArg == "" && fs.NArg() > 0 {
		nameArg = fs.Arg(0)
	}
	if nameArg == "" {
		fmt.Fprintln(os.Stderr, "error: environment name is required")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return exitToolError
	}

	filePath, resolvedName := env.ResolveEnvFile(cwd, nameArg)
	if filePath == "" {
		fmt.Fprintf(os.Stderr, "error: environment %q not found\n", nameArg)
		fmt.Fprintln(os.Stderr, "hint: run 'sailor env list' to see available environments")
		return exitToolError
	}

	vars, err := env.LoadEnvFile(filePath, resolvedName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	if jsonOutput {
		type jsonOut struct {
			Name   string            `json:"name"`
			Source string            `json:"source"`
			Vars   map[string]string `json:"vars"`
		}
		out := jsonOut{Name: nameArg, Source: filePath, Vars: vars}
		data, err := json.Marshal(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		fmt.Fprintln(os.Stdout, string(data))
		return exitOK
	}

	fmt.Fprintf(os.Stdout, "Environment: %s\n", nameArg)
	fmt.Fprintf(os.Stdout, "Source:      %s\n", filePath)
	fmt.Fprintln(os.Stdout)

	// Sort keys for stable output.
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	const keyW = 24
	for _, k := range keys {
		fmt.Fprintf(os.Stdout, "%-*s  %s\n", keyW, k, vars[k])
	}
	return exitOK
}
