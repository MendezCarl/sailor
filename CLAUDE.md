# CLAUDE.md

Instructions for Claude when working on this repository.

---

## Source of Truth

These documents define the intended design of the project. Read the relevant ones before making any change. If code conflicts with documentation, follow the documentation.

| Document | What it covers |
|---|---|
| `docs/product-spec.md` | Feature scope, philosophy, non-goals, target users |
| `docs/architecture.md` | Package structure, data flow, component responsibilities, exit codes |
| `docs/file-format.md` | YAML schemas, variable syntax, environment files, storage layout |
| `docs/roadmap.md` | Version milestones, what is in scope, what is permanently out of scope |
| `AGENTS.md` | Hard constraints, scope rules, contribution guidelines |

When a task is ambiguous, check these documents before proceeding. When a task conflicts with these documents, surface the conflict rather than resolving it silently.

---

## Project Overview

This is a lightweight, local-first, open-source CLI API client written in Go. It sends HTTP requests, manages saved request collections in YAML files, and supports environment-based variable substitution. It has no server component, no account system, and no cloud features. It compiles to a single static binary.

The project is in early development (v0.x). File formats and CLI interfaces are not yet stable.

---

## Repository Layout

```
README.md
AGENTS.md
CLAUDE.md
docs/
  product-spec.md
  architecture.md
  file-format.md
  roadmap.md
cmd/
  apitool/
    main.go
internal/
  cli/
  config/
  collection/
  env/
  request/
  executor/
  render/
  curl/
  plugin/
```

`cmd/` contains the binary entry point. `internal/` contains all application logic. Nothing outside `internal/` should contain business logic. Packages in `internal/` are not importable by external projects, which is intentional.

---

## Hard Rules

Do not do any of the following under any circumstances, regardless of what is requested:

- Add cloud sync, remote storage, or any outbound call that is not a user-initiated HTTP request
- Add login, accounts, registration, or any form of user identity
- Add telemetry, analytics, or crash reporting (if opt-in telemetry is ever added, it must be explicitly discussed first)
- Add team collaboration features
- Add AI integrations or calls to LLM APIs
- Add a server component, daemon, background process, or socket listener
- Add a startup network check, version check, or license validation
- Add GUI dependencies to the `cmd/` or `internal/` packages

If a task requires any of the above, stop and explain why it is out of scope rather than finding a workaround.

---

## Go Guidelines

### Standard library first

Before reaching for an external package, check whether the standard library solves the problem. `net/http`, `encoding/json`, `os`, `flag`, `text/template`, `bufio`, `strings`, `filepath`, and `io` cover a large fraction of what this tool needs. Use them.

When a dependency is genuinely necessary, add a comment to `go.mod` explaining why.

### Code style

- Format with `gofmt`. No exceptions.
- Prefer explicit error handling. Do not use `panic` outside of `main`.
- Prefer flat code over nested code. Early returns are preferable to deeply nested conditionals.
- Prefer explicit over clever. If a line of code requires a comment to explain what it does, rewrite the line.
- Do not add comments that restate what the code does. Add comments when the reason behind a decision is not obvious from the code itself.
- Do not add type annotations, docstrings, or comments to code you did not touch.

### Interfaces

Add an interface when you need to substitute implementations — typically for testing. Do not add interfaces speculatively. A function that takes a concrete type is simpler than one that takes an interface no one else implements.

### Error messages

Error messages are user-facing. Write them in lowercase, without punctuation at the end, and without package names or internal identifiers. Good: `request file not found: users.list`. Bad: `collection.Load: os.Open: no such file or directory`.

---

## Package Responsibilities

Each internal package has one job. Do not put logic in the wrong package.

| Package | Responsibility |
|---|---|
| `cli` | Parse flags and arguments, delegate to other packages. No business logic. |
| `config` | Load and merge global and project config files. Return a resolved `Config` struct. |
| `collection` | Read and parse collection YAML files. Resolve request names to file paths. |
| `env` | Load environment YAML files. Perform `${variable}` string interpolation. |
| `request` | Define the internal `Request` struct. All input paths normalize to this type. |
| `executor` | Execute a `Request` using `net/http`. Return a `Response`. No terminal output. |
| `render` | Format and write a `Response` to the terminal. No HTTP or file logic. |
| `curl` | Parse cURL strings into `Request` structs. Serialize `Request` structs to cURL strings. |
| `plugin` | Interface definitions only in v1. No plugin loading. |

If you are about to put HTTP logic in `render`, file logic in `executor`, or terminal output in `executor`, stop and reconsider. The data flow is linear:

```
input → cli → config → collection/env → request → executor → render → output
```

Each stage receives a typed input and produces a typed output. No stage skips another.

---

## CLI Conventions

- Response body goes to **stdout**. All other output — status lines, warnings, errors, timing — goes to **stderr**.
- `--raw` disables all formatting and decoration. The response body only, as-is, to stdout. Required for pipe compatibility.
- `--json` on meta-commands produces machine-readable output.
- Interactive prompts must have a flag equivalent. The tool must be scriptable in non-TTY environments.
- Do not change flag names, command names, or exit code semantics without checking `docs/architecture.md` and noting the change in a comment.

Exit codes are defined in `docs/architecture.md`. Do not add new exit codes without updating that document.

---

## Storage Conventions

All persistent files are YAML. Do not write JSON, TOML, SQLite, or binary formats for user-facing data.

Variable references use `${variable_name}` syntax. Do not introduce a different interpolation syntax.

Do not write absolute paths into any file. Do not embed machine-specific identifiers. All user files must be portable across machines via `git clone`.

Schema changes require updating `docs/file-format.md` before or alongside the code change. Do not change the schema silently.

---

## Performance Constraints

- Startup time target: under 100ms for `--version` or `--help` on modern hardware.
- Do not add initialization logic that runs unconditionally at startup if it is not needed for the current command.
- Do not load or parse files that the current command does not require.
- Do not buffer large response bodies in memory. Stream where the operation allows it.
- After adding a dependency, run `go build` and check the binary size delta. If size increases by more than ~500KB, justify it explicitly.

---

## Testing

- New behavior needs tests.
- Bug fixes need a regression test.
- Test observable behavior, not implementation details.
- Do not use mocking frameworks. Write interface-based mocks by hand when needed.
- Tests must be written to pass on Linux, macOS, and Windows. Avoid OS-specific assumptions.

---

## Scope Control

Before implementing any feature, ask:

1. **Does this make the tool better at sending HTTP requests and managing local collections?** If no, stop.
2. **Does it require a server, an account, or any network call other than user-initiated requests?** If yes, stop.
3. **Is it described in `docs/roadmap.md` as in-scope?** If not, raise it as a question before proceeding.
4. **What is the minimal implementation that satisfies the requirement?** Implement that, not more.

When a feature is possible but the cost in complexity is unclear, implement the minimal version and leave a `// TODO:` comment noting what a fuller version would require. Do not build speculatively.

---

## What Is In Scope for Future Work

The following are planned but not yet implemented. They are in scope when the roadmap reaches them. Do not implement them ahead of schedule without checking with the maintainers.

- GraphQL support — additive schema extension, no changes to the REST execution path
- WebSocket support — separate execution path under `apitool ws`, does not modify `executor`
- Request scripting — shell hooks only, no embedded scripting language, RFC required first
- Plugin system — subprocess model with JSON over stdio, no dynamic loading
- TUI — separate binary or entry point, shares file format and `executor`, does not affect CLI startup time

---

## When in Doubt

If a task is ambiguous, ask a clarifying question rather than making an assumption that could affect the file format, CLI interface, or package structure. These are the parts of the project most expensive to change later.

If a task conflicts with the documented design, say so explicitly. Do not silently work around a documented constraint. The right resolution might be to update the documentation — but that is a decision to make openly, not a shortcut to take quietly.
