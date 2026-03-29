# Roadmap

**Project:** (working title: api-testing-software)
**Last Updated:** 2026-03-28

This document describes the project's development direction, milestone goals, and long-term constraints. It is a living document — priorities may shift as the project matures and the community grows — but the core constraints listed in Section 6 are permanent and are not subject to revision.

---

## Table of Contents

1. [Project Status](#1-project-status)
2. [Versioning Strategy](#2-versioning-strategy)
3. [v0.x Goals](#3-v0x-goals)
4. [v1.0 Goals](#4-v10-goals)
5. [Post-v1 Roadmap](#5-post-v1-roadmap)
6. [Non-Goals (Permanent Constraints)](#6-non-goals-permanent-constraints)
7. [Performance Roadmap](#7-performance-roadmap)
8. [CLI → TUI → GUI Strategy](#8-cli--tui--gui-strategy)
9. [Plugin Ecosystem Vision](#9-plugin-ecosystem-vision)
10. [Community and Contribution Plan](#10-community-and-contribution-plan)
11. [Long-Term Vision](#11-long-term-vision)

---

## 1. Project Status

**Current phase: Early experimental development.**

The project does not yet have a stable release. APIs, CLI flags, and file formats are subject to breaking changes at any point during v0.x development. It is not recommended for use in production workflows or scripts that require stability.

What this means in practice:

- A CLI flag introduced in v0.2 may be renamed or removed in v0.3
- A YAML field in the collection format may change before v1.0
- Bugs may be fixed in ways that alter behavior

This is intentional. The goal during v0.x is to find the right design, not to lock in the wrong one. Stability is the goal of v1.0, not a prerequisite of v0.x.

---

## 2. Versioning Strategy

The project uses semantic versioning with a `v0.x` prefix until the core design is stable.

| Range | Meaning |
|---|---|
| `v0.1` – `v0.x` | Experimental. Breaking changes permitted between minor versions. |
| `v1.0` | Stable. CLI interface and file format are frozen. Semver enforced from this point. |
| `v1.x` | Additive changes only. No breaking changes to CLI or file format. |
| `v2.0+` | Only if a genuinely necessary breaking change cannot be avoided after exhausting alternatives. |

### What "breaking change" means

A breaking change is any change that causes an existing valid invocation or file to behave differently or fail. This includes:

- Renaming or removing a CLI flag
- Changing the meaning of a positional argument
- Removing or renaming a YAML field that is currently read
- Changing the variable interpolation syntax
- Changing exit code semantics

Adding new optional flags, new optional YAML fields, or new subcommands is never a breaking change.

### Pre-release labels

During v0.x, releases use simple minor version increments with no pre-release labels. There is no `-alpha`, `-beta`, or `-rc` suffix. Every v0.x release is implicitly unstable. The move to `v1.0` is the stability signal.

---

## 3. v0.x Goals

The v0.x phase is about building the right foundation. Feature breadth is explicitly not the goal. Each milestone should be fully stable before the next begins.

### v0.1 — Proof of concept

The minimum useful tool. Demonstrates that the core approach works.

- [ ] Send GET, POST, PUT, PATCH, DELETE requests from the CLI
- [ ] Set request headers via flags
- [ ] Set a request body from a flag or stdin
- [ ] Display response status, headers, and body
- [ ] Basic JSON response formatting in the terminal
- [ ] Exit code semantics defined and implemented
- [ ] Single binary build for Linux, macOS, and Windows (amd64)
- [ ] `--help` output on all commands

### v0.2 — File format foundation

Establishes the YAML formats that will stabilize into v1.0. This is the most important milestone to get right, because the file format is the user's long-term investment.

- [ ] Collection YAML format defined and documented
- [ ] Request YAML format defined and documented
- [ ] Environment YAML format defined and documented (single and multi-environment)
- [ ] `.env` file loading and variable interpolation
- [ ] `${variable}` syntax fully implemented
- [ ] `apitool run <name>` command for executing saved requests
- [ ] Global and project-local storage resolution
- [ ] `schema_version` field supported in all file types
- [ ] Format specification documented in `docs/file-format.md`

### v0.3 — Collections and environments

Makes the collection workflow practical for day-to-day use.

- [ ] Folder and group organization in collections
- [ ] `collection list` command
- [ ] `collection show <name>` command
- [ ] Environment list and inspection commands
- [ ] `--var` flag for per-invocation variable overrides
- [ ] Warning on undefined variable references
- [ ] cURL import: `apitool import curl`
- [ ] cURL export: `apitool export curl <name>`
- [ ] Arm64 binary targets (Apple Silicon, Linux arm64)

### v0.4 — Response experience

Polishes the terminal output to a level appropriate for daily use.

- [ ] Colorized status line with response time and size
- [ ] `--headers` / `-i` flag for showing response headers
- [ ] `--raw` flag for pipe-friendly output
- [ ] Long response paging via `$PAGER`
- [ ] `--no-pager` flag
- [ ] `--color=auto|always|never` flag
- [ ] `--fail-on-error` flag for scripting use cases
- [ ] Non-TTY detection with automatic color and decoration disable

### v0.5 — Configuration and ergonomics

Rounds out the configuration model and improves day-to-day ergonomics.

- [ ] Global config file (`config.yaml`) fully supported
- [ ] Project config override fully supported
- [ ] Config documentation in `docs/architecture.md`
- [ ] `--timeout` flag and config key
- [ ] `--follow-redirects` / `--no-follow-redirects`
- [ ] Request authentication helpers (Bearer, Basic, API key header construction)
- [ ] `--insecure` flag for self-signed certificates
- [ ] Consistent, well-documented error messages across all failure modes

### v0.6 — Hardening and cross-platform parity

No new features. Focus entirely on correctness, cross-platform behavior, and test coverage before the push toward v1.0.

- [ ] Full test coverage on the request execution pipeline
- [ ] Full test coverage on the YAML parser and variable interpolation engine
- [ ] Full test coverage on the cURL import/export round-trip
- [ ] Windows-specific terminal color handling verified
- [ ] Windows path separator handling verified
- [ ] All exit codes documented and tested
- [ ] Startup time benchmarked and regression tests in place
- [ ] No known data-loss or silent-failure bugs
- [ ] All `--help` output reviewed for accuracy

---

## 4. v1.0 Goals

v1.0 is a commitment. Once tagged, the CLI interface and file format are stable. Users can write scripts and commit collections without worrying about breakage on upgrade.

v1.0 is not a feature release. It is a stability release. The feature set entering v1.0 is whatever shipped in v0.x. No new features are added to reach v1.0 — only stabilization work.

### Stability criteria for v1.0

The following must be true before v1.0 is tagged:

**CLI interface**
- [ ] All commands, flags, and positional arguments are final
- [ ] No flags are deprecated or pending rename
- [ ] All `--help` text is accurate and complete
- [ ] Exit codes are documented in `docs/architecture.md` and will not change

**File format**
- [ ] Collection, request, environment, and config YAML schemas are final
- [ ] `schema_version: 1` is locked — any future breaking change requires `schema_version: 2`
- [ ] The format specification in `docs/file-format.md` exactly matches the parser's behavior
- [ ] A test suite validates all documented schemas

**Cross-platform**
- [ ] All five binary targets build and pass tests in CI: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`
- [ ] Behavior is identical across platforms for all documented features

**Performance**
- [ ] Startup time is under 100ms on the reference hardware defined in the performance roadmap (Section 7)
- [ ] Startup time regression tests are in CI

**Documentation**
- [ ] `README.md`, `docs/architecture.md`, `docs/file-format.md`, and `CONTRIBUTING.md` are complete and current
- [ ] A changelog documents all user-facing changes from v0.1 onward

**Quality**
- [ ] No open bugs tagged `blocker` or `regression`
- [ ] No known silent-failure or data-loss scenarios

### What v1.0 does not include

v1.0 ships exactly the features that proved stable during v0.x. Features not yet stabilized are deferred to v1.x releases. v1.0 will not include:

- GraphQL support (v1.x)
- WebSocket support (v1.x)
- Request scripting (v1.x)
- History file (v1.x, if the design stabilizes)
- Plugin loading (v1.x)
- TUI (v2.x)

---

## 5. Post-v1 Roadmap

Post-v1 development follows strict semver. All additions are backward-compatible with v1.0. No breaking changes without a v2.0 major version bump.

Priorities are listed in order. Later items will not be started until earlier items are complete and stable.

### Priority 1 — GraphQL support (v1.1)

GraphQL over HTTP is a POST request with a structured JSON body. The core execution pipeline requires no changes. New work is limited to:

- `graphql` block in the request YAML schema (query and variables)
- GraphQL error detection and separate rendering in the response view
- `apitool graphql introspect <url>` command for schema inspection

GraphQL support is purely additive. Existing REST collections are unaffected.

### Priority 2 — WebSocket support (v1.2)

WebSocket requires a different execution model than HTTP request/response. Implementation is a new `apitool ws` subcommand backed by a separate execution path:

- `apitool ws <url>` opens an interactive connection
- `--message` flag for single-send non-interactive mode
- Streaming output for received frames
- Saved WebSocket sessions in the collection YAML format

WebSocket support does not modify the REST or GraphQL execution paths.

### Priority 3 — Request scripting (v1.3)

Lightweight hooks that allow shell commands to run before or after a request, enabling simple chaining and extraction without requiring a full scripting language:

- `hooks.pre` and `hooks.post` fields in the request YAML schema
- Response data injected as environment variables to hook scripts
- No embedded scripting language — hooks are shell commands only
- Disabled by default; opt-in per request

The full design will be proposed via the RFC process before implementation begins.

### Priority 4 — TUI (v2.0)

A terminal user interface for users who prefer an interactive workflow. The TUI is a separate binary target, not a mode of the existing CLI. It shares the same file format, config, and execution engine.

The TUI milestone is a major version because it requires significant new infrastructure. It does not break the CLI. See Section 8 for the full CLI → TUI → GUI strategy.

### Priority 5 — GUI (long-term, optional)

A lightweight native GUI for developers who prefer a visual interface. This is explicitly the lowest priority item on the post-v1 roadmap and will only be pursued if the project has the contributors and resources to build it without compromising the CLI and TUI.

The GUI is never a replacement for the CLI — it is an additional surface that shares the same local storage and execution core. See Section 8.

---

## 6. Non-Goals (Permanent Constraints)

The following items are not deferred or deprioritized — they are excluded permanently. They will not be added regardless of demand, popularity, or contributor interest. Any pull request implementing these features will be closed.

| Feature | Reason |
|---|---|
| **Cloud sync** | Requires a server. This tool has no server and will never have one. |
| **User accounts or login** | The tool has no concept of user identity. There is no authentication layer to build on. |
| **Team collaboration features** | Git is the collaboration layer. This tool treats collections as files, not shared state. |
| **AI-assisted features** | Adds external dependencies, cost, and a new class of privacy concerns that are incompatible with the tool's design. |

These constraints exist because they define what this tool is. A tool with cloud sync, accounts, team features, and AI is a different product — one that already exists. This project's value is precisely that it is none of those things.

If you want to build a tool that includes these features, that is a valid project. It is not this project.

---

## 7. Performance Roadmap

Performance is a first-class feature, not an afterthought. The following milestones define measurable performance goals and protect them through regression testing.

### Reference hardware

Performance targets are measured on:

- A mid-range laptop (defined as a machine approximately equivalent to a 2021 MacBook Pro M1 or a mid-tier x86_64 Linux laptop with 8GB RAM)
- A cold start (binary not in OS disk cache)
- A warm start (binary in OS disk cache, measured for typical interactive use)

### Startup time targets

| Milestone | Target | Status |
|---|---|---|
| v0.1 | Under 200ms cold start | — |
| v0.4 | Under 150ms cold start | — |
| v0.6 | Under 100ms cold start | — |
| v1.0 | Under 100ms cold start, regression test in CI | — |
| Post-v1 | Maintain under 100ms as features are added | — |

Startup time is measured from process invocation to first byte of output for a `apitool --version` command. This is the baseline that all feature work must not regress.

### Binary size targets

| Milestone | Target |
|---|---|
| v1.0 | Under 15MB for all targets |
| Post-v1 | Maintain under 20MB including GraphQL and WebSocket |

Binary size is a proxy for dependency discipline. A growing binary without a corresponding user-visible feature addition is a signal that a dependency was added carelessly.

### Memory usage targets

| Operation | Target |
|---|---|
| Startup (idle) | Under 10MB RSS |
| Single request execution | Under 20MB RSS |
| Large response body (10MB) | Under 50MB RSS, streamed where possible |

### Dependency policy enforcement

- The dependency list is reviewed at every minor version bump
- Any dependency added must be accompanied by a justification comment in `go.mod`
- Any dependency that increases binary size by more than 500KB requires explicit discussion in the PR
- The total number of direct dependencies is tracked and included in the changelog

---

## 8. CLI → TUI → GUI Strategy

The project expands its interface surface in three phases, each separated by significant development work. Later phases do not replace earlier ones — they add surfaces.

### Phase 1: CLI (current and permanent)

The CLI is the primary interface and will remain fully supported indefinitely regardless of what other surfaces are added. Every feature available in the TUI or GUI is also available in the CLI. The CLI is the reference implementation.

Users who write scripts, work in CI, or prefer terminal workflows will never be asked to migrate to a different interface.

### Phase 2: TUI (post-v1)

The TUI is an interactive terminal interface — tabs, panels, keyboard navigation — built using a Go terminal UI framework (Bubble Tea is the current candidate). It is not a mode of the CLI binary; it is a separate entry point in the same repository that compiles to its own binary or a named subcommand.

The TUI shares:
- The same YAML file format
- The same config and environment system
- The same request execution engine (`internal/executor`)
- The same response rendering logic where applicable

The TUI does not have:
- A different storage format
- A different concept of collections or environments
- Any cloud or sync features

A user who switches between CLI and TUI workflows should find that their collections, environments, and config work identically in both.

### Phase 3: GUI (optional, long-term)

The GUI is a lightweight native application — not Electron — intended for developers who strongly prefer a visual interface. It is the lowest priority and will only be built if resources allow.

Constraints that apply to the GUI regardless of when it is built:

- It must use the same file format as the CLI and TUI — no separate storage or sync
- It must not require an account or any network connection to function
- It must not be the only way to access any feature — every GUI action must also be possible via CLI
- It should be built as a thin interface over the same execution core, not a new implementation

---

## 9. Plugin Ecosystem Vision

Plugin support is not implemented in v1. This section describes the intended design so that v1 architecture decisions do not foreclose it.

### Design principles for plugins

- Plugins are external executables, not dynamically loaded libraries
- Communication is via stdin/stdout using newline-delimited JSON
- Any language that can read and write JSON to stdio can write a plugin
- The core binary remains unaffected by plugins at startup — plugin loading is lazy
- Plugins cannot modify core behavior — they extend it at defined extension points

### Extension points (intended, not yet implemented)

| Point | What a plugin can do |
|---|---|
| Pre-request interceptor | Modify the request before it is sent (add headers, sign requests, inject tokens) |
| Post-request transformer | Annotate or modify the response before rendering |
| Output formatter | Replace the default response renderer with a custom format |

### Plugin discovery

In v1.x, plugins are discovered by scanning the `plugins/` directory of the global config. There is no registry, no marketplace, and no hosted index. A community-maintained list in the repository's wiki is the intended discovery mechanism.

### What plugins will not be allowed to do

- Make outbound network calls without explicit user acknowledgment
- Write to files outside the user's config directory
- Store credentials or secrets
- Modify the collection or environment files directly

Plugin security boundaries will be documented before the plugin runtime is implemented. Contributions to the plugin design are welcome through the RFC process.

---

## 10. Community and Contribution Plan

### Current phase: Foundations

During v0.x, the primary contribution need is core implementation work. The architecture is still being shaped, and contributions that involve large structural changes are best coordinated through issues before a pull request is opened.

**Good first issues** during v0.x include:

- Improving `--help` text and error messages
- Adding test coverage to the parser or execution pipeline
- Fixing cross-platform behavior differences
- Writing or correcting documentation
- Reporting and reproducing bugs

### RFC process (planned for v0.5+)

For any change that affects the CLI interface, the file format, or the plugin interface, the project will use a lightweight RFC (Request for Comments) process:

1. Open a GitHub issue with the `rfc` label describing the proposed change
2. Allow a comment period (minimum two weeks for format changes, one week for CLI changes)
3. Summarize the discussion and document the decision in the issue
4. Implement the accepted design

This process exists to prevent the file format and CLI interface from being shaped by whoever happens to submit a PR first. Decisions that affect stability should be deliberate.

### Architecture milestones

The following architectural improvements are planned for the v0.x phase to make contribution easier:

| Milestone | Goal |
|---|---|
| v0.2 | Package boundaries clearly defined, each package independently testable |
| v0.4 | Interface-based design in `executor` and `render` to allow unit testing without network calls |
| v0.6 | Contribution guide (`CONTRIBUTING.md`) fully written with code style, test requirements, and PR process |
| v1.0 | Architecture documentation (`docs/architecture.md`) matches the implementation exactly |

### Contributor onboarding (planned for v0.6+)

Before v1.0, the project will provide:

- A `CONTRIBUTING.md` covering setup, build, test, and PR process
- A `docs/architecture.md` that explains the package structure well enough that a new contributor can navigate the codebase without asking questions
- A set of `good-first-issue` labeled issues kept current and unassigned
- A code of conduct

---

## 11. Long-Term Vision

In the long term, this tool should be:

- The default choice for developers who want a fast, local, CLI-native API client
- A stable enough foundation that other tools can build on the file format
- A well-maintained open-source project with a small, active contributor community
- Exactly as large as it needs to be and no larger

The project will be considered mature when:

- The file format has been stable for at least two years
- The CLI interface has had no breaking changes since v1.0
- The codebase is small enough that a new contributor can understand the entire thing in a day
- The binary size and startup time are the same or better than they were at v1.0

The project will be considered off-track if:

- Feature requests regularly cite what Postman or Insomnia can do as justification
- The dependency count has grown significantly without a proportional improvement to users
- Contributors are regularly proposing accounts, sync, or collaboration features
- Startup time has regressed without a corresponding user-visible feature improvement

The purpose of this roadmap is not to predict the future — it is to stay honest about what the project is for. A fast, local, free CLI API client. Nothing more, and nothing less.
