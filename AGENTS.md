# AGENTS.md

This file defines rules and guidelines for AI agents and automated contributors working in this repository. Read it before making any change.

When in doubt about whether a change is appropriate, consult the documents in `/docs` before proceeding. The documentation is the source of truth. If code conflicts with documentation, the documentation wins.

---

## 1. Purpose

This project is a lightweight, local-first, open-source CLI API client. It is intentionally small. Every architectural decision, dependency choice, and feature addition should serve that purpose — or it should not happen.

AI agents working in this repository are expected to preserve the project's character, not just its code. A technically correct change that violates the project's philosophy is a bad change.

---

## 2. Project Philosophy

These are the values that drive every decision in this project. Treat them as constraints, not preferences.

| Value | What it means in practice |
|---|---|
| **Lightweight** | The binary is small. The code is small. The feature set is small. |
| **CLI-first** | The terminal interface is the primary and permanent interface. |
| **Local-first** | All data lives on the user's machine. No remote storage, ever. |
| **Open-source** | The project is fully open. No closed components, no proprietary formats. |
| **Always free** | No paid tiers. No paywalled features. No hosted services to monetize. |
| **Privacy-focused** | The tool makes no outbound calls other than user-initiated requests. |
| **Minimal dependencies** | Every dependency is justified. The standard library is preferred. |
| **Single binary** | The tool ships as one compiled file with no runtime dependencies. |
| **Git-friendly** | All user data is plain text, diffable, and committable. |
| **Performance over features** | A fast, limited tool is better than a slow, complete one. |

---

## 3. Hard Constraints

The following are permanent project constraints. They are not subject to revision, community vote, or user demand. Do not implement, partially implement, scaffold, or leave hooks for any of these.

### Never add

- **Cloud sync** — The tool has no server component and will never have one. Do not add any code that uploads, syncs, or mirrors user data to a remote service.
- **User accounts or login** — There is no concept of user identity in this tool. Do not add authentication flows, session tokens, registration, or any mechanism that identifies a user to a remote system.
- **Team collaboration features** — Collections are files. Git is the collaboration layer. Do not add shared workspaces, team permissions, invite flows, or any multi-user state.
- **AI features** — Do not add LLM integrations, AI-assisted request generation, smart suggestions, or any feature that calls an AI API.
- **Telemetry** — Do not add usage analytics, error reporting, crash collection, or any outbound call that is not a user-initiated HTTP request. If opt-in telemetry is ever added, it must be: explicitly disabled by default, clearly documented, and must only send data to infrastructure the user controls.
- **Server-side components** — Do not add a companion server, a relay service, a hosted config endpoint, or any backend. The tool is a single binary that makes network calls only to URLs the user provides.
- **Required internet connectivity** — The tool must be fully functional offline. Do not add version checks, update prompts, license validation, or any startup behavior that requires a network connection.

If a feature requires any of the above to function, the feature is out of scope.

---

## 4. Architecture Rules

### Language

The implementation language is **Go**. Do not introduce files written in other languages that become part of the build or runtime. Shell scripts for tooling are acceptable. Embedded runtimes (Node.js, Python, WASM) are not.

### Binary output

The tool must compile to a single static binary on all supported platforms. Do not add dependencies that require CGO unless there is no alternative and the need is explicitly discussed in a GitHub issue first. Do not add runtime dependencies that must be installed separately.

Verify that `go build ./...` produces a working binary with no external dependencies after any change you make.

### Cross-platform

All changes must work correctly on Linux, macOS, and Windows. This includes:

- File path handling — use `filepath.Join`, not string concatenation with `/`
- Terminal color — ANSI codes require explicit enablement on Windows; do not assume they work
- Config directories — use `os.UserConfigDir()`, not hardcoded `~/.config`
- Line endings — write files with `\n`; do not assume the OS default

Do not merge a change that has only been considered on one platform.

### Package structure

The codebase is organized under `internal/`. Keep packages focused and flat. Do not introduce deep package hierarchies. Do not create a package for a single function. Do not create shared utility packages that accumulate unrelated helpers.

Each package should have a single, describable responsibility. If you cannot describe what a package does in one sentence, it should be split or merged.

### Dependencies

Before adding any external dependency:

1. Check whether the standard library can solve the problem. If it can, use it.
2. If a dependency is necessary, verify that it has no transitive dependencies that phone home, embed telemetry, or make network calls.
3. Run `go build` before and after to measure the binary size impact.
4. Add a comment to `go.mod` explaining why the dependency is needed.

Do not add a dependency to avoid writing 20 lines of straightforward code. Do not add a dependency because it is convenient. Convenience is not a justification.

---

## 5. CLI Design Rules

### Command structure

Commands follow a flat verb-noun pattern. Do not introduce deeply nested subcommand trees. Do not create a subcommand where a flag would work. Do not create a flag where a positional argument would work.

### Predictability

The tool must behave predictably. Do not add behavior that changes based on implicit state, environment detection, or heuristics unless the behavior is clearly documented and overridable with a flag.

If something is happening automatically, it must be visible in the output.

### Composability

- Response body goes to **stdout**
- Diagnostics, status lines, and warnings go to **stderr**
- A `--raw` flag must disable all decoration and output only the response body
- Meta-commands must support a `--json` flag for machine-readable output
- The tool must be fully operable in non-TTY environments (CI, scripts, pipes)

Do not add interactive prompts that cannot be bypassed with a flag.

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Request sent and response received (any HTTP status) |
| `1` | Tool error (bad input, missing file, config error) |
| `2` | Network error (connection refused, timeout, DNS failure) |
| `3` | HTTP 4xx/5xx (only when `--fail-on-error` is set) |

Do not change these without updating `docs/architecture.md` and the test suite.

### No hidden state

The tool must not maintain background processes, daemon connections, or persistent in-memory state between invocations. Each invocation is independent. Do not add a local server, a socket listener, or a lock file that persists between runs.

---

## 6. Storage Rules

### Format

All persistent data is stored as **YAML**. Do not introduce JSON, TOML, SQLite, binary formats, or any other format for user-facing files. Do not store user data in a database.

### Human readability

Every file the tool writes must be readable and editable by a developer who has never used this tool. Do not write files that require the tool to be understood. Do not use YAML features that obscure meaning: no anchors, no aliases, no custom tags.

### Git-friendliness

Files must produce clean diffs. Do not write files that sort keys randomly, embed timestamps in locations that cause noise, or include volatile metadata that changes on every write when the logical content has not changed.

### Portability

Do not write absolute paths into any file. Do not embed machine-specific identifiers. A file written on one machine must work identically on another after a `git clone`.

### Schema changes

Do not change the YAML schema for requests, collections, environments, or config in a way that breaks existing files without:

1. Incrementing `schema_version`
2. Documenting the migration path in `docs/file-format.md`
3. Maintaining backward compatibility for at least one version

If you are unsure whether a schema change is breaking, it is breaking.

---

## 7. Performance Rules

Performance is a feature. Do not regress it.

### Startup time

The tool must start in under **100ms** on modern hardware for a `--version` or `--help` invocation. This is a hard target, not a guideline. Do not add initialization logic that runs unconditionally at startup. Do not load files, make system calls, or parse config that is not needed for the current command.

### Memory

The tool should use under **10MB RSS** at startup. Response body processing should stream where possible rather than loading entire bodies into memory. Do not buffer large responses into a single in-memory allocation unless the operation requires it.

### Abstractions

Do not add abstraction layers that exist only to be abstract. Every interface, wrapper, and indirection adds cognitive overhead and often adds runtime overhead. Add an interface when you need to substitute implementations (e.g., for testing). Do not add one because it feels like good architecture.

### Benchmarks

If a change touches the request execution path, the YAML parser, or the variable interpolation engine, include a benchmark. Do not claim a change has no performance impact without measuring it.

---

## 8. Scope Control Rules

### Before implementing any feature, ask

1. Does this make the tool better at **sending HTTP requests and managing local collections**?
2. Does it require a server, an account, or a network connection to function?
3. Does it increase binary size or startup time without a proportional user benefit?
4. Is it described in the product spec (`docs/product-spec.md`) or roadmap (`docs/roadmap.md`)?

If the answer to (1) is no, or the answer to (2) is yes, stop.

### Reject or flag the following

Do not implement and do not leave scaffolding for:

- Features that add a GUI dependency to the core CLI binary
- Features that add cloud, sync, or remote storage
- Features that require user authentication
- Features that embed a scripting engine, template language, or expression evaluator more complex than simple `${variable}` string substitution
- Features that are primarily useful in a team or multi-user context
- Features that add startup latency for users who do not use them

### Complexity budget

This project has a complexity budget. Every feature spends from it. A feature that touches five packages, adds two new YAML fields, and requires a new dependency costs more than a feature that adds a flag to an existing command. Spend the budget carefully.

When a feature is possible but not obviously worth its complexity cost, do not add it. Leave a comment or open an issue. Let maintainers decide.

---

## 9. Contribution Rules

### Documentation is the source of truth

The following documents define the intended design of this project:

| Document | What it governs |
|---|---|
| `docs/product-spec.md` | Feature scope, philosophy, user personas |
| `docs/architecture.md` | Package structure, data flow, component responsibilities |
| `docs/file-format.md` | YAML schemas, variable syntax, storage layout |
| `docs/roadmap.md` | Version milestones, post-v1 priorities, permanent non-goals |

If your change conflicts with these documents, either update the document first (through an issue and discussion) or do not make the change. Do not silently diverge from documented design.

### Code style

- Prefer explicit over clever
- Prefer readable over compact
- Prefer flat over nested
- Follow standard Go formatting (`gofmt`)
- Do not add comments that restate what the code does — add comments that explain why, when the reason is not obvious
- Do not add type annotations, docstrings, or comments to code you did not change

### Testing

- New behavior requires tests
- Bug fixes require a regression test that would have caught the bug
- Tests must pass on all three platforms
- Do not use test mocking frameworks — write interface-based mocks by hand
- Do not test implementation details — test observable behavior

### Changes that require discussion before implementation

Open a GitHub issue before writing code for:

- Any change to the YAML file format
- Any change to CLI flag names, command names, or exit code semantics
- Any new external dependency
- Any change to the package structure
- Any feature not described in the current roadmap

Small changes (bug fixes, error message improvements, test additions, documentation corrections) do not require pre-approval.

---

## 10. Future Expansion Rules

The following features are planned for future versions. They are in scope but not yet being built. Do not implement them ahead of schedule without explicit coordination with maintainers.

| Feature | Target | Notes |
|---|---|---|
| GraphQL support | v1.1 | Additive schema extension. REST core must be stable first. |
| WebSocket support | v1.2 | Separate execution path. Does not modify HTTP pipeline. |
| Request scripting | v1.3 | Shell hooks only. No embedded scripting engine. RFC required. |
| TUI | v2.0 | Separate binary or entry point. Shares file format and executor. |
| GUI | Long-term | Native only (not Electron). Separate from CLI core. Optional. |
| Plugin system | v1.x | Subprocess model, JSON over stdio. No dynamic loading. |

### Rules for future expansion work

- The **CLI core must remain unaffected** by TUI or GUI work. Adding a TUI must not increase CLI startup time or binary size.
- The **GUI must never be required** to access any feature. Every GUI action must also be available via CLI.
- The **plugin interface** must be defined and stable before any plugins are built. Do not build a plugin runtime before the interface is finalized through the RFC process.
- **GraphQL and WebSocket** are additive. Implementing them must not require changes to the REST execution path. If they do, the design is wrong.

### Rules that do not change with version

Regardless of how the project evolves, the following rules apply to every version:

- No cloud component
- No accounts
- Single binary distributable
- All user data in local plain-text files
- No required network connectivity for the tool itself
- Free forever
