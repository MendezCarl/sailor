package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/MendezCarl/sailor.git/internal/collection"
)

// runCollection dispatches `sailor collection <subcommand>`.
func runCollection(args []string, cwd string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "Usage: sailor collection <subcommand>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Subcommands:")
		fmt.Fprintln(os.Stderr, "  list   List all collections in the current project")
		fmt.Fprintln(os.Stderr, "  show   Show details of a named collection")
		return 0
	}
	switch args[0] {
	case "list":
		return runCollectionList(args[1:], cwd)
	case "show":
		return runCollectionShow(args[1:], cwd)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown collection subcommand %q\n\n", args[0])
		fmt.Fprintln(os.Stderr, "Subcommands: list, show")
		return exitToolError
	}
}

// runCollectionList implements `sailor collection list`.
func runCollectionList(args []string, cwd string) int {
	fs := flag.NewFlagSet("collection list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor collection list [--json]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Lists all collections in .apitool/collections/ in the current directory.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitToolError
	}

	dir := collection.DefaultCollectionDir(cwd)
	entries, err := collection.ListAll(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "no collections found in %s\n", dir)
		fmt.Fprintln(os.Stderr, "hint: add collection YAML files to .apitool/collections/")
		return exitToolError
	}

	if jsonOutput {
		type jsonEntry struct {
			Name         string `json:"name"`
			File         string `json:"file"`
			RequestCount int    `json:"request_count"`
		}
		out := make([]jsonEntry, len(entries))
		for i, e := range entries {
			out[i] = jsonEntry{
				Name:         e.Collection.Name,
				File:         e.FilePath,
				RequestCount: collection.CountRequests(e.Collection),
			}
		}
		data, err := json.Marshal(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		fmt.Fprintln(os.Stdout, string(data))
		return exitOK
	}

	// Human-readable table.
	const nameW = 30
	for _, e := range entries {
		n := collection.CountRequests(e.Collection)
		noun := "requests"
		if n == 1 {
			noun = "request"
		}
		fmt.Fprintf(os.Stdout, "%-*s  %s  (%d %s)\n",
			nameW, e.Collection.Name, e.FilePath, n, noun)
	}
	return exitOK
}

// runCollectionShow implements `sailor collection show <name>`.
func runCollectionShow(args []string, cwd string) int {
	fs := flag.NewFlagSet("collection show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		collectionFile string
		jsonOutput     bool
	)
	fs.StringVar(&collectionFile, "collection", "", "path to a collection YAML file")
	fs.BoolVar(&jsonOutput, "json", false, "output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sailor collection show <name> [--collection <file>] [--json]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Shows details of a named collection.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Flags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  sailor collection show \"User API\"")
		fmt.Fprintln(os.Stderr, "  sailor collection show --collection examples/users.yaml")
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

	var col *collection.Collection
	if collectionFile != "" {
		var err error
		col, err = collection.LoadFile(collectionFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
	} else {
		if nameArg == "" {
			fmt.Fprintln(os.Stderr, "error: collection name is required")
			fmt.Fprintln(os.Stderr)
			fs.Usage()
			return exitToolError
		}
		dir := collection.DefaultCollectionDir(cwd)
		entries, err := collection.ListAll(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		for _, e := range entries {
			if e.Collection.Name == nameArg {
				col = e.Collection
				break
			}
		}
		if col == nil {
			fmt.Fprintf(os.Stderr, "error: collection %q not found in %s\n", nameArg, dir)
			return exitToolError
		}
	}

	if jsonOutput {
		type jsonReq struct {
			Name   string `json:"name"`
			Method string `json:"method"`
			URL    string `json:"url"`
			Folder string `json:"folder"`
		}
		type jsonCol struct {
			Name        string    `json:"name"`
			Description string    `json:"description,omitempty"`
			BaseURL     string    `json:"base_url,omitempty"`
			Requests    []jsonReq `json:"requests"`
		}
		var reqs []jsonReq
		for _, r := range col.Requests {
			reqs = append(reqs, jsonReq{Name: r.Name, Method: r.Method, URL: r.URL, Folder: ""})
		}
		var collectFolderReqs func(folders []*collection.Folder, prefix string)
		collectFolderReqs = func(folders []*collection.Folder, prefix string) {
			for _, f := range folders {
				p := prefix
				if p != "" {
					p += "."
				}
				p += f.Name
				for _, r := range f.Requests {
					reqs = append(reqs, jsonReq{Name: r.Name, Method: r.Method, URL: r.URL, Folder: p})
				}
				collectFolderReqs(f.Folders, p)
			}
		}
		collectFolderReqs(col.Folders, "")
		out := jsonCol{
			Name:        col.Name,
			Description: col.Description,
			BaseURL:     col.BaseURL,
			Requests:    reqs,
		}
		data, err := json.Marshal(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitToolError
		}
		fmt.Fprintln(os.Stdout, string(data))
		return exitOK
	}

	// Human-readable output.
	fmt.Fprintf(os.Stdout, "Name:         %s\n", col.Name)
	if col.Description != "" {
		fmt.Fprintf(os.Stdout, "Description:  %s\n", col.Description)
	}
	if col.BaseURL != "" {
		fmt.Fprintf(os.Stdout, "Base URL:     %s\n", col.BaseURL)
	}
	total := collection.CountRequests(col)
	fmt.Fprintf(os.Stdout, "Requests (%d):\n", total)

	for _, r := range col.Requests {
		fmt.Fprintf(os.Stdout, "  %-30s  %-6s  %s\n", r.Name, r.Method, r.URL)
	}
	var printFolders func(folders []*collection.Folder, indent string)
	printFolders = func(folders []*collection.Folder, indent string) {
		for _, f := range folders {
			fmt.Fprintf(os.Stdout, "%s[%s]\n", indent, f.Name)
			for _, r := range f.Requests {
				fmt.Fprintf(os.Stdout, "%s  %-28s  %-6s  %s\n", indent, r.Name, r.Method, r.URL)
			}
			printFolders(f.Folders, indent+"  ")
		}
	}
	printFolders(col.Folders, "  ")

	return exitOK
}
