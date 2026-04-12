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
)

// pipelineOpts holds all inputs for a single request execution cycle:
// variable collection → interpolation → HTTP execution → rendering.
type pipelineOpts struct {
	EnvDir      string
	CLIVars     env.Vars
	BaseVars    env.Vars
	RenderOpts  render.Options
	Config      *config.Config
	CLITimeout  string
	FailOnError bool
	EnvFilePath string
	EnvName     string
	ExecOpts    executor.Options
}

// runRequestPipeline applies variables, executes the request, and renders output.
// Returns an exit code.
func runRequestPipeline(req *request.Request, opts pipelineOpts) int {
	vars, err := env.Collect(opts.EnvDir, opts.BaseVars, opts.CLIVars, opts.EnvFilePath, opts.EnvName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	resolved, err := env.Apply(req, vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	timeout, err := resolveTimeout(opts.CLITimeout, resolved, opts.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	resp, err := executor.Send(resolved, timeout, opts.ExecOpts)
	if err != nil {
		var netErr *executor.NetworkError
		if errors.As(err, &netErr) {
			fmt.Fprintf(os.Stderr, "error: could not connect: %s\n", err)
			return exitNetworkError
		}
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return exitToolError
	}

	render.Print(os.Stdout, os.Stderr, resp, opts.RenderOpts)

	if opts.FailOnError && resp.StatusCode >= 400 {
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
