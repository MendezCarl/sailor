# Architecture

**Project:** Sailor
**Version:** 0.1 — MVP
**Status:** Draft
**Last Updated:** 2026-03-28

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Design Goals](#2-design-goals)
3. [Technology Choice Rationale](#3-technology-choice-rationale)
4. [CLI Interaction Model](#4-cli-interaction-model)
5. [Core System Components](#5-core-system-components)
6. [Request Execution Flow](#6-request-execution-flow)
7. [Local Storage and Config Model](#7-local-storage-and-config-model)
8. [YAML File Strategy](#8-yaml-file-strategy)
9. [Global vs Project Configuration](#9-global-vs-project-configuration)
10. [Response Rendering in the Terminal](#10-response-rendering-in-the-terminal)
11. [cURL Import/Export Architecture](#11-curl-importexport-architecture)
12. [Future GraphQL and WebSocket Extension Path](#12-future-graphql-and-websocket-extension-path)
13. [Plugin Architecture Placeholder](#13-plugin-architecture-placeholder)
14. [Cross-Platform Considerations](#14-cross-platform-considerations)
15. [Dependency Philosophy](#15-dependency-philosophy)
16. [Security and Privacy Model](#16-security-and-privacy-model)
17. [Non-Goals and Scope Guardrails](#17-non-goals-and-scope-guardrails)

---

## 1. Architecture Overview

This tool is a CLI-first HTTP client implemented as a single compiled Go binary. It has no server-side component, no runtime dependencies, no account system, and no network requirements beyond executing the user's own HTTP requests.

The architectural topology is intentionally flat:

```
User
  └── CLI binary (single process)
        ├── Command parser
        ├── Config loader (global + project)
        ├── Collection/environment loader (YAML files on disk)
        ├── Request builder
        ├── HTTP executor (net/http)
        └── Response renderer (terminal output)
```

There is no daemon, no background process, no IPC layer, and no local server. The binary starts, executes a command, writes output, and exits. This is the complete process model for v1.

All persistent state — saved requests, collections, environments, and configuration — is stored as human-readable YAML files on the local filesystem. These files are designed to be committed to Git, edited by hand, and transferred between machines without any tooling other than a file copy.

---

## 2. Design Goals

These goals are ordered by priority. When two goals conflict, the higher-ranked goal wins.

| Priority | Goal | Implication |
|---|---|---|
| 1 | Single binary distribution | No runtime, no installer, no daemon |
| 2 | Local-first, no server component | All state on disk; no network calls to use the tool itself |
| 3 | Privacy — no telemetry, no accounts | No outbound calls except user-initiated requests |
| 4 | Fast startup and execution | Prefer compile-time work over runtime work |
| 5 | Git-friendly storage | Plain text, stable diffs, no binary formats |
| 6 | Composability with Unix tools | Pipeable output, standard exit codes, non-interactive mode |
| 7 | Contributor readability | Flat structure, minimal abstraction, standard library first |
| 8 | Cross-platform parity | Same behavior and binary quality on Windows, macOS, Linux |

Any architectural decision that serves a lower-priority goal at the expense of a higher-priority goal requires explicit justification and community discussion before being accepted.

---

## 3. Technology Choice Rationale

### 3.1 Go

Go is the implementation language for the following reasons:

**Single binary output.** `go build` produces a statically linked binary by default. There is no runtime to install, no interpreter, no shared library dependencies on most targets. A user downloads one file and runs it.

**Cross-platform compilation.** Go's cross-compilation support is first-class. Building for `GOOS=windows GOARCH=amd64`, `GOOS=darwin GOARCH=arm64`, and `GOOS=linux GOARCH=amd64` from a single machine requires no additional toolchains.

**Strong standard library.** Go's `net/http`, `encoding/json`, `os`, `flag`, and `text/template` packages cover the majority of what this tool needs. Reaching for external packages should be the exception, not the default.

**Startup performance.** Go binaries start in milliseconds. The tool's target startup budget is under 100ms on modern hardware. A JVM, Node.js, or Python runtime would make this target difficult to meet consistently.

**Readability for contributors.** Go's syntax is intentionally limited. There is typically one idiomatic way to write a given construct. This lowers the cognitive overhead for contributors who are new to the codebase.

**Explicit error handling.** Go's explicit error returns make the control flow of a CLI tool transparent. Silent failures are difficult to accidentally write.

### 3.2 YAML for Storage

YAML is chosen for request files, collections, and environments over JSON and TOML for the following reasons:

- Multi-line strings (request bodies, GraphQL queries) are readable without escaping
- Comments are supported, enabling inline documentation in saved requests
- Human editability without special tooling is high
- Git diffs are clean and meaningful

The tradeoff — YAML parsing is more complex than TOML, and YAML has footguns — is accepted. The file schema will be kept intentionally simple to avoid YAML's more problematic features (anchors, complex types, implicit type coercion).

### 3.3 No Framework for CLI Parsing

The CLI interface will use either Go's standard `flag` package or a single, well-scoped CLI library (such as `cobra`). If a library is used, it must not pull in a large dependency tree and must not impose a code organization pattern that conflicts with the flat structure described in Section 5.

The decision between `flag` and `cobra` should be based on whether subcommand support is needed from the start. If it is, `cobra` is acceptable. If commands can be structured as positional arguments or a simple verb-noun scheme, the standard library is preferred.

---

## 4. CLI Interaction Model

### 4.1 Two Primary Workflows

The CLI supports two distinct usage patterns:

**One-off commands** — Send a request directly from the command line without any saved state:

```sh
# GET request
sailor get https://api.example.com/users

# POST with a JSON body
sailor post https://api.example.com/users --json '{"name": "alice"}'

# POST with a body from a file
sailor post https://api.example.com/users --body ./payload.json

# With explicit headers
sailor get https://api.example.com/users -H "Authorization: Bearer $TOKEN"
```

**Collection-based workflow** — Execute named requests from a saved local collection:

```sh
# Run a named request from the current project's collection
sailor run users.list

# Run with a specific environment
sailor run users.list --env staging

# Run with a variable override
sailor run users.create --var name=alice
```

Both workflows produce identical output. The collection workflow is syntactic sugar over the same request execution pipeline — a named YAML file is loaded and resolved into the same internal request structure that a one-off command produces directly.

### 4.2 Command Structure

Commands follow a flat verb-noun pattern. Subcommands are avoided where a simple flag achieves the same result. The command surface should be learnable in a single session.

```
sailor <verb> [target] [flags]

Verbs:
  get, post, put, patch, delete, head     HTTP method shortcuts
  run <name>                               Execute a saved request
  import curl <curl-string>               Import a cURL command as a saved request
  export curl <name>                       Export a saved request as a cURL command
  env list                                 List available environments
  collection list                          List saved requests in the current collection
  collection show <name>                   Print a saved request file
```

### 4.3 Composability Requirements

The CLI must behave as a well-behaved Unix tool:

- **Stdout for data, stderr for diagnostics.** Response body goes to stdout. Status messages, warnings, and progress indicators go to stderr. This allows `sailor get ... | jq` to work without noise.
- **`--raw` flag.** Disables formatting, syntax highlighting, and all decorations. Outputs the raw response body only.
- **`--json` flag on meta-commands.** Commands like `collection list` and `env list` should support `--json` to produce machine-readable output.
- **Exit codes.** Described in full in Section 6. Tool errors are non-zero. HTTP responses are zero unless `--fail-on-error` is set.
- **Non-interactive mode.** Any interactive prompt must have a flag equivalent. The tool must be fully scriptable in environments with no TTY.

---

## 5. Core System Components

The codebase is organized around a small set of focused packages. The structure is intentionally flat. Deep package hierarchies add navigation overhead without adding clarity.

```
cmd/
  apitool/
    main.go               Entry point. Parses top-level command and delegates.

internal/
  cli/                    Command definitions, flag parsing, help text
  config/                 Config loading: global, project, merge logic
  collection/             YAML file loading, request resolution, collection listing
  env/                    Environment file loading, variable interpolation
  request/                Internal request model, builder, validator
  executor/               HTTP execution via net/http, timeout handling
  render/                 Terminal output: colorization, formatting, paging
  curl/                   cURL string parsing and export serialization
  plugin/                 Plugin interface definition (placeholder, v1 stub)

```

### 5.1 Component Responsibilities

**`cli`** — Owns the user-facing command surface. Parses flags, validates inputs, and delegates to internal packages. Does not contain business logic. Thin wrappers over `config`, `collection`, `executor`, and `render`.

**`config`** — Loads and merges the global config file and the project-local config override. Exposes a single resolved `Config` struct to the rest of the application. Does not perform I/O beyond file reads.

**`collection`** — Reads YAML files from the collection directory. Resolves request names to file paths. Unmarshals request YAML into the internal `Request` model. Does not execute requests.

**`env`** — Loads named environment YAML files. Performs variable interpolation on request fields using a simple `{{variable}}` syntax. Warns on undefined variables. Does not store secrets.

**`request`** — Defines the internal `Request` struct. This is the canonical representation of an HTTP request throughout the application. All input paths — one-off commands, collection files, cURL imports — normalize to this struct.

**`executor`** — Takes a `Request` struct and executes it using `net/http`. Returns a `Response` struct. Handles timeouts, redirects (configurable), and connection errors. Has no knowledge of terminal output or file formats.

**`render`** — Takes a `Response` struct and writes formatted output to the terminal. Responsible for colorization, syntax highlighting, header display, response metadata (timing, size), and paging. Has no knowledge of HTTP or file formats.

**`curl`** — Parses cURL command strings into `Request` structs. Serializes `Request` structs back to cURL command strings. Isolated from all other concerns.

**`plugin`** — Defines the `Plugin` interface and documents the future extension points. In v1, this package contains only type definitions and comments. No plugin loading occurs.

### 5.2 Data Flow Summary

```
Input (CLI flags / YAML file / cURL string)
  └── [cli] Parse and validate
        └── [config] Load merged config
              └── [collection + env] Resolve named request + apply variables
                    └── [request] Build internal Request struct
                          └── [executor] Execute via net/http → Response struct
                                └── [render] Write formatted output to terminal
```

Every stage has a single, well-defined input type and output type. No stage skips another. This makes each component independently testable.

---

## 6. Request Execution Flow

This section describes what happens between the user pressing enter and output appearing on the terminal.

### 6.1 Step-by-Step

1. **Parse command.** The `cli` package parses the subcommand, positional arguments, and flags. Invalid input produces a usage error to stderr and exits non-zero immediately.

2. **Load config.** The `config` package loads the global config file and, if a project config exists in the current working directory or any parent directory, merges it. The project config overrides the global config for any key it defines.

3. **Resolve request.**
   - For one-off commands: flags are mapped directly to a `Request` struct.
   - For `run` commands: the named request is located in the collection directory, the YAML file is read, and it is unmarshaled into a `Request` struct.

4. **Apply environment.** If an `--env` flag is provided or a default environment is configured, the corresponding environment YAML file is loaded. Variable references in the `Request` struct fields (URLs, headers, body) are interpolated. Undefined variables produce a warning; they are not silently left as-is.

5. **Apply overrides.** Any `--var`, `-H`, or other per-invocation overrides are applied to the resolved `Request` struct after environment interpolation.

6. **Validate.** The request is checked for required fields (method, URL). Basic URL validation is performed. This is not deep validation — it is a sanity check to catch obvious mistakes before making a network call.

7. **Execute.** The `executor` package sends the request using `net/http`. The response — status code, headers, body, and timing metadata — is captured into a `Response` struct. Connection errors, timeouts, and DNS failures produce a tool-level error (non-zero exit).

8. **Render.** The `render` package writes the response to stdout (body) and stderr (diagnostics). If stdout is not a TTY, colorization is disabled automatically. If `--raw` is set, only the response body is written, with no decoration.

9. **Exit.** The process exits `0` if the request was sent and a response was received, regardless of HTTP status code. The process exits non-zero only on tool-level failures (no response received, config error, file not found). If `--fail-on-error` is set, HTTP 4xx and 5xx responses also cause non-zero exit.

### 6.2 Exit Code Definitions

| Code | Meaning |
|---|---|
| `0` | Request sent and response received (any HTTP status) |
| `1` | General tool error (config invalid, file not found, flag error) |
| `2` | Network error (connection refused, DNS failure, timeout) |
| `3` | HTTP 4xx or 5xx response (only when `--fail-on-error` is set) |

---

## 7. Local Storage and Config Model

### 7.1 Directory Layout

The tool uses two storage locations:

**User-global storage** (`~/.config/sailor/` on Linux/macOS, `%APPDATA%\sailor\` on Windows):

```
~/.config/sailor/
  config.yaml           Global user config (timeout, default env, output prefs)
  envs/
    default.yaml        Default environment variables
    staging.yaml        Named environment
    production.yaml     Named environment
  collections/
    personal/           User's personal saved requests
      users.list.yaml
      users.create.yaml
  plugins/              Plugin binaries (v2+, empty in v1)
```

**Project-local storage** (relative to the project root, committed to Git):

```
<project-root>/
  .apitool/
    config.yaml         Project-level config overrides
    envs/
      local.yaml        Local dev environment (may be gitignored)
      shared.yaml       Shared non-secret config (committed)
    collections/
      api/              API request collection for this project
        auth.login.yaml
        users.list.yaml
        users.create.yaml
```

The project root is determined by walking up the directory tree from the current working directory until a `.apitool/` directory is found, matching the same discovery pattern used by Git.

### 7.2 Collection Naming Convention

Request files are named using dot-separated path notation that mirrors their filesystem location:

```
collections/
  users/
    list.yaml       → users.list
    create.yaml     → users.create
  auth/
    login.yaml      → auth.login
```

The `run` command uses this dot notation: `sailor run users.list`. The tool resolves the name to a file path by replacing dots with directory separators and appending `.yaml`.

---

## 8. YAML File Strategy

### 8.1 Request File Schema

A single request file is a self-contained, human-readable description of an HTTP request:

```yaml
# users.list.yaml
name: List Users
method: GET
url: "{{base_url}}/users"

headers:
  Authorization: "Bearer {{auth_token}}"
  Accept: application/json

params:
  page: "1"
  limit: "20"

# body is omitted for GET requests
```

```yaml
# users.create.yaml
name: Create User
method: POST
url: "{{base_url}}/users"

headers:
  Authorization: "Bearer {{auth_token}}"
  Content-Type: application/json

body: |
  {
    "name": "{{user_name}}",
    "email": "{{user_email}}"
  }
```

All fields except `method` and `url` are optional. The schema is deliberately minimal.

### 8.2 Environment File Schema

An environment file defines variable values for a named environment:

```yaml
# envs/staging.yaml
base_url: https://staging.api.example.com
auth_token: ""        # set via override or local secret file
request_timeout: 30s
```

```yaml
# envs/local.yaml  (gitignored, contains secrets)
base_url: http://localhost:8080
auth_token: dev-token-abc123
```

### 8.3 Config File Schema

```yaml
# config.yaml (global or project)
default_env: local
timeout: 30s
follow_redirects: true
max_redirects: 5
output:
  color: auto           # auto | always | never
  format: pretty        # pretty | raw | json
  show_headers: false   # show response headers by default
  pager: auto           # auto | always | never
```

### 8.4 Schema Versioning

All YAML files include an optional `schema_version` field. In v1, the schema version is `1`. When the schema changes in a breaking way, the version increments and a migration path is documented. The tool reads the version field and emits a clear warning if it encounters a file with an unrecognized version rather than silently misreading it.

---

## 9. Global vs Project Configuration

Configuration is resolved through a two-level merge. Project config overrides global config. No other levels exist.

### 9.1 Resolution Order

```
Global config   (~/.config/sailor/config.yaml)
      +
Project config  (.apitool/config.yaml, if present)
      =
Resolved config (used for the current invocation)
```

The merge is shallow: any key present in the project config replaces the corresponding key from the global config entirely. Deep merging of nested objects is not performed — this keeps the merge behavior predictable and easy to reason about.

### 9.2 Environment Resolution Order

When an environment is requested (via `--env` flag or `default_env` config key), it is resolved in this order:

1. Project-local environment file: `.apitool/envs/<name>.yaml`
2. Global environment file: `~/.config/sailor/envs/<name>.yaml`
3. Error if neither exists

This allows a project to define its own `staging` environment that overrides or supplements a global one.

### 9.3 Collection Resolution Order

When a named request is referenced (via `sailor run <name>`):

1. Project-local collection: `.apitool/collections/<name>.yaml`
2. Global collection: `~/.config/sailor/collections/<name>.yaml`
3. Error if neither exists

### 9.4 Recommended Git Workflow

```
.apitool/
  config.yaml         ✓ commit  — project defaults, no secrets
  envs/
    shared.yaml       ✓ commit  — non-secret shared variables (base URLs, feature flags)
    local.yaml        ✗ ignore  — secrets, local overrides (add to .gitignore)
  collections/        ✓ commit  — all request files, treat as project assets
```

The convention: anything without a secret value is safe to commit. Developers create a `local.yaml` for secrets and add it to `.gitignore`. A `shared.yaml` contains the non-secret baseline that all team members can use immediately after cloning.

This is a convention, not enforcement. The tool does not scan for secrets or block commits. Documentation and `.gitignore` templates are the primary mechanism.

---

## 10. Response Rendering in the Terminal

### 10.1 Output Layers

Response output is composed of distinct layers, each independently controllable:

```
┌─────────────────────────────────────┐
│  Status line                        │  HTTP/1.1 200 OK  (32ms, 1.4kb)
├─────────────────────────────────────┤
│  Response headers  (opt-in)         │  Content-Type: application/json
│                                     │  X-Request-Id: abc123
├─────────────────────────────────────┤
│  Response body     (always)         │  { "users": [...] }
└─────────────────────────────────────┘
```

- **Status line** is always shown on stderr when connected to a TTY. It includes the HTTP status code and text, response time in milliseconds, and response size in human-readable units.
- **Response headers** are hidden by default and shown with `--headers` or `-i`.
- **Response body** is always written to stdout. When the body is JSON and stdout is a TTY, it is pretty-printed and syntax-highlighted.

### 10.2 Color and Formatting

Color output uses ANSI escape codes. Color is applied only when stdout is connected to a TTY (`os.Stdout.Fd()` is a terminal). When piping output, color is automatically disabled without requiring a flag.

The `--color` flag overrides this: `--color=always` forces color even in pipes; `--color=never` disables it even in a TTY. The default is `auto`.

JSON syntax highlighting uses a small, purpose-built colorizer rather than a heavy third-party library. The required color set is minimal: string values, numeric values, keys, punctuation. A full-featured syntax highlighting library is not warranted for this use case.

### 10.3 Diff-Friendly Output

When `--raw` is set, the output is the response body with no decoration, no color, and no extra newlines beyond what the server sent. This mode is designed for:

- Piping to `jq`, `grep`, `diff`, or other tools
- Capturing response bodies in shell scripts
- Comparing responses between environments

For explicit response comparison, a `--save <name>` flag can persist the last response to a file in the collection directory. Users can then `diff` the files themselves using standard tools. The tool does not implement a built-in diff command in v1.

### 10.4 Paging Long Responses

When the response body is larger than the terminal height, the tool pipes the output to the system's configured pager (`$PAGER`, defaulting to `less`). This only applies when stdout is a TTY. In pipe mode, the full output is written without paging.

The `--no-pager` flag disables paging regardless of response size.

### 10.5 Response Metadata

The status line includes:

- HTTP version, status code, and status text
- Response time (measured from first byte sent to last byte received)
- Response body size (Content-Length if available, otherwise measured)

This metadata is written to stderr, not stdout, so it does not contaminate piped output.

---

## 11. cURL Import/Export Architecture

cURL interoperability is a first-class feature. The ability to import a cURL command from documentation or a browser's DevTools and immediately have a saved request — and conversely, to export any saved request as a cURL command to share with a colleague — is a core workflow.

### 11.1 Internal Request Model as the Canonical Form

The `Request` struct in the `request` package is the single source of truth for what an HTTP request looks like in this tool. Both cURL import and cURL export operate through this struct:

```
cURL string  →  [curl.Parse]  →  Request struct  →  [curl.Serialize]  →  cURL string
YAML file    →  [collection.Load]  →  Request struct
CLI flags    →  [cli.Parse]  →  Request struct
```

This means cURL round-trips are reliable: import a cURL command, export it again, and the result should be semantically equivalent. It also means that fixing the internal `Request` model automatically improves both import and export behavior.

### 11.2 cURL Parser

The cURL parser handles the subset of cURL flags that appear in documentation, browser export, and common usage:

| cURL flag | Meaning |
|---|---|
| `-X`, `--request` | HTTP method |
| `-H`, `--header` | Request header |
| `-d`, `--data`, `--data-raw` | Request body |
| `--data-binary` | Binary body (treated as raw data) |
| `--json` | Body with `Content-Type: application/json` |
| `-u`, `--user` | Basic auth credentials |
| `-b`, `--cookie` | Cookie header |
| `--compressed` | Accept-Encoding: gzip (informational only) |
| `-L`, `--location` | Follow redirects |
| `--max-redirs` | Max redirect count |
| `-k`, `--insecure` | Skip TLS verification |
| `-s`, `--silent` | Silent mode (ignored on import) |
| `-o`, `--output` | Output file (ignored on import) |

Flags not in this list are ignored with a warning rather than causing a parse failure. The goal is tolerant import, not strict cURL compatibility.

### 11.3 cURL Serializer

The serializer converts a `Request` struct to a minimal, valid cURL command string. The output uses long flag names (`--header`, not `-H`) for readability and is formatted across multiple lines using `\` continuation for requests with multiple headers or a body.

```sh
curl --request POST \
  --url 'https://api.example.com/users' \
  --header 'Authorization: Bearer token123' \
  --header 'Content-Type: application/json' \
  --data '{"name": "alice"}'
```

Variable references (`{{base_url}}`) in the `Request` struct are preserved as-is in the cURL export. The user is expected to resolve variables before sharing.

### 11.4 Import Command

```sh
# From a string
sailor import curl "curl -X POST https://api.example.com/users -H 'Content-Type: application/json' -d '{\"name\":\"alice\"}'"

# From a file containing a cURL command
sailor import curl --file ./curl-command.txt

# Import and immediately save to collection
sailor import curl "..." --save users.create
```

When `--save` is not provided, the imported request is printed to stdout as a YAML request file. The user can inspect and save it manually.

---

## 12. Future GraphQL and WebSocket Extension Path

GraphQL and WebSocket support are deferred until the REST workflow is stable. This section documents the intended extension path so that v1 design decisions do not accidentally foreclose these protocols.

### 12.1 GraphQL

GraphQL over HTTP is mechanically a POST request with a JSON body containing `query`, `variables`, and optionally `operationName`. The core execution pipeline requires no changes. GraphQL-specific additions are:

- **Request file schema extension:** A `graphql` block in the YAML schema that holds the query and variables separately, rendered as a JSON body at execution time.
- **Response rendering:** GraphQL responses contain an `errors` array at the top level. The renderer should detect and highlight these separately from HTTP errors.
- **Introspection shortcut:** A `sailor graphql introspect <url>` command to fetch and display a schema.

The `executor` package requires no changes for GraphQL. The `render` package requires a GraphQL-aware response detector. The `collection` package requires schema support for the `graphql` block.

### 12.2 WebSocket

WebSocket support requires a persistent connection model that is architecturally different from the request/response model used for HTTP. It should be implemented as a distinct subcommand (`sailor ws <url>`) backed by a separate execution path, not grafted onto the existing `executor` package.

Key extension points:

- A `ws` package that wraps `golang.org/x/net/websocket` or a minimal WebSocket library
- An interactive read/write loop that reads user input from stdin and writes received frames to stdout
- A `--message` flag for non-interactive single-send mode (send one message, print response, exit)
- A separate YAML schema for saved WebSocket sessions (URL, headers, initial messages)

The `render` package will need a streaming output mode for WebSocket frames, but this is additive and does not require restructuring existing rendering logic.

### 12.3 Keeping v1 Clean

The v1 codebase should not contain any stub code for GraphQL or WebSocket beyond the `plugin` placeholder described in Section 13. Extension happens by adding new packages, not by modifying the core request pipeline. The `Request` struct should not grow GraphQL or WebSocket fields until those protocols are actively being implemented.

---

## 13. Plugin Architecture Placeholder

Plugin support is not implemented in v1. This section defines where it will live and what it will look like so that v1 architectural decisions do not need to be undone.

### 13.1 Extension Points

Three natural extension points exist in the request lifecycle:

| Point | What a plugin could do |
|---|---|
| Pre-request (request interceptor) | Modify the `Request` struct before execution (add headers, sign requests) |
| Post-request (response transformer) | Modify or annotate the `Response` struct before rendering |
| Output formatter | Replace the default renderer with a custom format |

### 13.2 Plugin Model (Intended, v2+)

Plugins will be external executables placed in the `plugins/` directory of the global config. The tool invokes them as subprocesses and communicates via stdin/stdout using a JSON protocol over newline-delimited messages.

This model is chosen over dynamic loading (`.so` files, WASM) because:

- It requires no plugin SDK in a specific language — any language that can read and write JSON to stdio works
- It isolates plugin crashes from the main process
- It avoids CGO and dynamic linking complications on cross-platform builds
- It keeps the core binary free of plugin-related complexity until plugins are actually being used

### 13.3 v1 Stub

The `plugin` package in v1 contains:

```go
// plugin/plugin.go

// Plugin defines the interface that future plugin implementations will satisfy.
// No plugins are loaded or executed in v1.
type Plugin interface {
    // InterceptRequest is called before request execution.
    // It may return a modified Request or the original unchanged.
    InterceptRequest(req *request.Request) (*request.Request, error)

    // TransformResponse is called after request execution.
    // It may return a modified Response or the original unchanged.
    TransformResponse(res *executor.Response) (*executor.Response, error)
}
```

This package is not wired into the execution pipeline in v1. Its existence documents intent and allows contributors to design against the interface before the runtime is built.

---

## 14. Cross-Platform Considerations

The tool targets Windows, macOS, and Linux with equal priority. The following platform-specific concerns are addressed explicitly.

### 14.1 Config and Data Directories

The tool uses platform-appropriate directories via Go's `os.UserConfigDir()` and `os.UserCacheDir()`:

| Platform | Config directory |
|---|---|
| Linux | `~/.config/sailor/` |
| macOS | `~/Library/Application Support/sailor/` |
| Windows | `%APPDATA%\sailor\` |

The tool does not hardcode `~/.config` on any platform.

### 14.2 Terminal Color Support

ANSI color codes work on macOS and Linux terminals natively. On Windows, ANSI support is available in Windows Terminal and recent versions of PowerShell and cmd.exe (Windows 10 1903+), but must be explicitly enabled via `windows.EnableVirtualTerminalProcessing`. The render package handles this at startup and falls back gracefully to no-color mode if the call fails.

### 14.3 Path Separators

All internal path handling uses `filepath.Join` and `filepath.ToSlash` rather than string concatenation with `/`. YAML files that contain file paths use forward slashes, which Go normalizes on Windows automatically.

### 14.4 Line Endings

YAML files are written with Unix line endings (`\n`). Git's `text=auto` attribute in `.gitattributes` handles platform normalization for users who clone on Windows. The tool does not normalize line endings in response bodies.

### 14.5 Shell and Quoting

The tool does not invoke a shell. It does not call `exec.Command("sh", "-c", ...)` with user-provided input. Commands are executed directly via `exec.Command` when invoking plugins (v2+), using argument arrays rather than shell strings. This eliminates an entire class of injection vulnerabilities.

### 14.6 Build Matrix

CI must build and test on all three platforms. The release pipeline produces the following binary targets as a minimum:

```
sailor-linux-amd64
sailor-linux-arm64
sailor-darwin-amd64
sailor-darwin-arm64
sailor-windows-amd64.exe
```

---

## 15. Dependency Philosophy

### 15.1 The Default Answer is No

When evaluating whether to add a dependency, the default answer is no. The burden of proof is on the dependency, not on the standard library alternative.

A dependency is acceptable when:

1. The standard library genuinely cannot accomplish the task without significant reimplementation effort
2. The dependency is small, well-maintained, and has a stable API
3. The dependency does not pull in a large transitive graph
4. The dependency does not make network calls, embed telemetry, or have a non-obvious runtime behavior

### 15.2 Evaluation Criteria

Before adding any dependency, evaluate it against:

- **Size:** What does it add to the binary? Run `go build` before and after.
- **Transitive dependencies:** Run `go mod graph`. A dependency that pulls in 20 others is a 21-dependency decision.
- **Maintenance status:** Is it actively maintained? Does it have a history of breaking changes?
- **Scope creep:** Does it solve only the problem at hand, or does it encourage patterns that expand scope?
- **Network behavior:** Does it check for updates, send analytics, or contact any external service?

### 15.3 Permitted Dependency Categories

| Category | Acceptable? | Notes |
|---|---|---|
| CLI framework (`cobra`, `urfave/cli`) | Conditionally | Only if standard `flag` is insufficient; evaluate at project start |
| YAML parsing (`gopkg.in/yaml.v3`) | Yes | Standard library has no YAML support |
| Terminal color (`fatih/color`, similar) | Conditionally | Only if ANSI handling across platforms cannot be done without it |
| WebSocket (v2+) | Yes | Standard library has no WebSocket support |
| HTTP client | No | `net/http` is sufficient |
| JSON parsing | No | `encoding/json` is sufficient |
| Template rendering | No | `text/template` is sufficient |
| Test mocking frameworks | No | Use interface-based mocks written by hand |

### 15.4 Vendoring

Dependencies are vendored (`go mod vendor`) in the repository. This ensures reproducible builds without network access and makes the full dependency tree auditable by reading the `vendor/` directory. It also prevents supply-chain attacks from package registry compromises during CI builds.

---

## 16. Security and Privacy Model

### 16.1 No Outbound Calls from the Tool Itself

The tool makes no network connections other than those explicitly initiated by the user via request commands. There are no:

- Update checks
- Telemetry pings
- Crash reporters
- License validation calls
- Analytics events

This is verifiable by code review. Any contribution that adds an outbound network call not initiated by a user request will not be accepted.

### 16.2 No Persistent Credentials

The tool does not store credentials in an OS keychain, a database, or an encrypted store. Environment files are plain text on disk. Users manage their own secret hygiene using the `.gitignore` convention described in Section 9.4.

This is a deliberate v1 constraint, not an oversight. A future version may add optional keychain integration, but it will be strictly opt-in and will not change the default behavior.

### 16.3 No Account System

There is no login, no registration, no session token, no OAuth flow for the tool itself. These concepts do not exist in the architecture. The `auth` terminology in this codebase refers exclusively to request-level authentication helpers (Bearer token construction, Basic auth header encoding) — never to user authentication.

### 16.4 TLS Verification

TLS certificate verification is enabled by default. It can be disabled with `--insecure` for development against self-signed certificates. The flag is explicit and visible; it is not a configuration file option that could be quietly set and forgotten.

### 16.5 Secret Leakage Prevention

Response bodies and request bodies are written to stdout but are not logged to any file by default. The `--verbose` flag writes request and response details to stderr but does so visibly. No silent logging occurs.

When `--save` persists a response to disk, the file is written only to the path the user specifies, in the directory the user controls. The tool does not write response data anywhere else.

### 16.6 Input Handling

The tool does not evaluate user-supplied data as code. Template interpolation (`{{variable}}`) is a simple string substitution performed by the `env` package. It does not support expressions, function calls, or any form of evaluation. A template value resolves to a string or it does not resolve. There is no execution context.

---

## 17. Non-Goals and Scope Guardrails

This section is addressed to future contributors. The items below are not missing features — they are deliberate boundaries. Before implementing anything adjacent to this list, reread the product spec and consider whether the addition serves the tool's core user or expands the target to someone else.

### 17.1 Explicit Non-Goals

| Item | Architectural Reason |
|---|---|
| Cloud sync | Requires a server component, which this tool does not have and will not have |
| User accounts or sign-in | No concept of a user identity exists in the system |
| Team collaboration features | Git is the collaboration layer; the tool does not duplicate it |
| AI-assisted request generation | Adds external dependency, cost, and a new class of privacy concerns |
| API monitoring or uptime alerts | A different product requiring persistent processes and notifications |
| Mock server | Would require an HTTP server component and a separate lifecycle |
| API documentation hosting | Requires a server and a publishing workflow |
| Built-in test assertions | Shell scripting and `jq` compose with `--raw` output; a test framework adds scope |
| GUI or desktop application | CLI-first; a GUI is a future consideration requiring a separate architectural decision |
| Plugin marketplace or registry | Requires hosted infrastructure and moderation |

### 17.2 Scope Escalation Signals

The following are warning signs that a proposed change may be escalating scope beyond the tool's purpose:

- The change requires a new network connection from the tool itself (not the user's request)
- The change requires storing state in a format other than plain YAML files
- The change requires user authentication or identity
- The change makes the binary significantly larger without a proportional improvement in the core workflow
- The change adds a background process or daemon
- The change requires a new external service to be useful

None of these automatically disqualify a contribution. They are signals that the change needs explicit discussion against the product principles before implementation begins.

### 17.3 The Scope Test

Before implementing a new feature, apply this test:

> Does this feature make the tool better at sending HTTP requests and managing request collections locally, without requiring any external service or user account?

If the answer is yes, it is likely in scope. If the answer requires qualification — "well, it makes the tool better at X, but you need to have Y configured" — it is likely out of scope for v1 and should be proposed as a future item with a clear rationale.
