# Product Specification

**Project:** Sailor
**Version:** 0.1 — MVP
**Status:** Draft
**Last Updated:** 2026-03-28

---

## Table of Contents

1. [Overview](#1-overview)
2. [Problem Statement](#2-problem-statement)
3. [Product Vision](#3-product-vision)
4. [Core Principles](#4-core-principles)
5. [Target Users](#5-target-users)
6. [Primary Use Cases](#6-primary-use-cases)
7. [MVP Scope](#7-mvp-scope)
8. [Non-Goals](#8-non-goals)
9. [CLI-First UX Principles](#9-cli-first-ux-principles)
10. [Storage and Privacy Principles](#10-storage-and-privacy-principles)
11. [Differentiators](#11-differentiators)
12. [Future Expansion Notes](#12-future-expansion-notes)

---

## 1. Overview

This tool is a lightweight, local-first, open-source HTTP client for developers. It is designed for the command line and built around the idea that testing and exploring APIs should be fast, private, scriptable, and require no account, no subscription, and no internet connection to operate.

It supports REST, GraphQL, and WebSocket requests. Collections are plain files stored on disk. Environment variables are text files. Everything is designed to live in a Git repository alongside the code that uses it.

The tool is free. It will always be free.

---

## 2. Problem Statement

API clients like Postman were once simple utilities. Over time they have become bloated platforms requiring accounts, offering cloud sync, promoting team workspaces, and pushing AI features. The free tiers of these tools are increasingly restricted. Their desktop apps are slow Electron applications. Their storage is increasingly cloud-dependent. Privacy expectations are unclear.

Developers who want to test a single endpoint or maintain a collection of requests for a local project are forced to log in, accept privacy policies, wait for slow UIs to load, and navigate feature-heavy interfaces designed for enterprise teams rather than individual contributors.

The core problem is simple: **there is no well-maintained, fast, open-source, CLI-native API client that treats the developer's local machine as the only place that matters.**

Existing CLI alternatives such as `curl` and `httpie` are excellent low-level tools but lack collection management, environment variable support, multi-protocol handling, and a structured workflow for iterative API exploration. They are building blocks, not complete workflows.

This tool fills that gap.

---

## 3. Product Vision

A developer should be able to clone a repository, run a single binary, and immediately send requests to any API — without creating an account, without connecting to any external service, and without reading a manual.

Request collections, environment configurations, and history should all live as plain text files in the same directory as the project they belong to. They should be readable by humans, diffable by Git, and portable across machines via version control.

The tool should feel closer to a well-designed CLI utility than to an application. It should start instantly, respond instantly, and get out of the way.

The long-term vision is a tool that developers trust precisely because of what it does not do: it does not phone home, it does not require sign-in, it does not sync anything, and it does not grow beyond its core purpose without deliberate and community-driven decisions.

---

## 4. Core Principles

These principles are not preferences — they are constraints. Future contributors should treat any feature that conflicts with them as out of scope, regardless of demand.

### 4.1 Local-First

All data lives on the user's device. Collections, environments, history, and configuration are files on disk. No network call is ever required to use the tool itself. The tool operates fully offline except when executing the user's own HTTP requests.

### 4.2 No Accounts

There is no sign-in, no registration, no user profile, no authentication to the tool itself. The concept of a user account does not exist in this product. Any reference to "auth" in this document refers exclusively to request-level authentication mechanisms (e.g., Bearer tokens, Basic auth headers) — never to app-level user accounts.

### 4.3 Privacy by Design

The tool collects no telemetry, no usage data, no analytics, and no crash reports unless the user explicitly opts in via a local configuration flag and the data is sent nowhere outside the user's own infrastructure. There is no hosted backend. There is no server that receives anything. The authors of this tool cannot see anything a user does with it.

### 4.4 Open Source

The tool is open source. The license must allow free use, modification, and redistribution. The codebase should be readable and approachable for contributors at all experience levels.

### 4.5 Performance Over Features

Startup time, response rendering, and command execution must be fast. A slow feature is a bad feature. An unimplemented feature is preferable to a sluggish one. Benchmarks for startup time and request execution should be tracked and protected as the codebase grows.

### 4.6 Git-Friendly Storage

All persistent state is stored in plain text formats (JSON, TOML, or YAML — one format, consistently). File structures should be stable, human-readable, and produce clean, meaningful diffs. Collections and environments should be committable to version control without modification.

### 4.7 Minimal Dependencies

The dependency tree should be as small as justifiable. Every dependency is a maintenance liability and a potential supply-chain risk. Prefer standard library implementations. Justify each external dependency explicitly.

### 4.8 Single Binary Distribution

The primary distribution target is a single compiled binary with no runtime dependencies. Users should be able to download one file and run it immediately on any supported platform.

### 4.9 Free Forever

This tool will not introduce paid tiers, premium features, or subscription requirements. It will not be acquired and paywalled. The commitment to being free is architectural: by having no server infrastructure, there is no cost basis that would pressure monetization.

---

## 5. Target Users

The tool is designed for developers who work with APIs regularly and want a focused, fast, local workflow. It is not designed for non-technical users, product managers, or QA teams using visual test builders.

### 5.1 Primary Users

| User Type | Context |
|---|---|
| Backend developers | Building and debugging REST and GraphQL APIs |
| Full-stack developers | Testing endpoints during local development |
| Frontend developers | Exploring third-party APIs, debugging integration issues |
| DevOps engineers | Scripting health checks, validating infrastructure endpoints |

### 5.2 Skill Level

The tool should be approachable for developers who are comfortable with a terminal but new to API testing workflows. It should not require deep knowledge of HTTP internals to use basic features. Advanced users should find it scriptable and composable with other CLI tools.

---

## 6. Primary Use Cases

### 6.1 Ad Hoc Request Execution

A developer needs to quickly send a `POST` request with a JSON body to a local development server. They run a single command, see the response, and move on. No GUI, no project setup, no account.

### 6.2 Saved Request Collections

A developer maintains a collection of requests for their project's API. The collection is a directory of files checked into the same repository as the code. When a new team member clones the repo, they have immediate access to all saved requests.

### 6.3 Environment-Based Configuration

A developer switches between local, staging, and production environments by specifying a different environment file. Base URLs, API keys, and other variables are defined per-environment and referenced in requests. No environment-specific data is hardcoded in collection files.

### 6.4 cURL Interoperability

A developer finds a cURL command in documentation or a Stack Overflow answer. They import it directly and it becomes a saved request. Conversely, they export any saved request as a cURL command to share with a colleague or include in documentation.

### 6.5 GraphQL Exploration

A developer queries a GraphQL API, sends introspection queries, and iterates on query structure from the terminal.

### 6.6 WebSocket Testing

A developer opens a persistent WebSocket connection, sends messages, and observes responses in a streaming view within the terminal.

### 6.7 Scripted Workflows

A developer pipes the tool's output into `jq`, writes shell scripts that call it in sequence, or integrates it into a CI pipeline to run smoke tests against a deployed environment.

---

## 7. MVP Scope

The MVP establishes the core workflow. It does not attempt to be feature-complete. Every item listed here must be stable and fast before any additional features are considered.

### 7.1 Request Execution

- Send HTTP requests using GET, POST, PUT, PATCH, DELETE, HEAD, and OPTIONS methods
- Set request headers as key-value pairs
- Set a request body (raw text, JSON, form data)
- Display response status code, headers, and body
- Display response time and response size
- Format and syntax-highlight JSON response bodies in the terminal

### 7.2 REST Support

- Full support for standard HTTP methods and headers
- Support for query parameters
- Support for path variables resolved from environment files

### 7.3 GraphQL Support

- Send GraphQL queries and mutations via HTTP POST
- Support for GraphQL variables
- Detect and display GraphQL errors separately from HTTP errors

### 7.4 WebSocket Support

- Open a persistent WebSocket connection
- Send messages interactively from the terminal
- Display incoming messages in a streaming view
- Close the connection explicitly or on interrupt signal

### 7.5 Environment Variables

- Define named environment files as plain text (TOML or JSON)
- Reference variables in request URLs, headers, and bodies using a standard interpolation syntax (e.g., `{{base_url}}`)
- Switch active environment via a command flag
- Warn on undefined variable references at execution time

### 7.6 Collections and Folders

- Organize saved requests into named collections stored as a directory structure on disk
- Each request is a single file
- Collections are portable — moving or copying the directory is sufficient to transfer them
- No database, no binary formats, no lock-in

### 7.7 cURL Import and Export

- Parse a cURL command string and create a saved request from it
- Export any saved request as a valid cURL command
- Support common cURL flags: `-X`, `-H`, `-d`, `-u`, `--data-raw`, `--json`

### 7.8 Response Viewer

- Display response body with syntax highlighting for JSON and plain text
- Page long responses using the terminal's native pager
- Support a raw output mode for piping to other tools
- Display headers separately from the body when requested

### 7.9 Multi-Request Workflow

- Named sessions or a request history file allow re-executing and referencing prior requests within a session
- A `--last` or equivalent flag re-runs the most recent request
- Named saved requests can be executed by name without re-specifying parameters

### 7.10 Plugin System Placeholder

- Define a documented, stable interface for plugins before v1 ships
- Plugins are executable files placed in a known directory and invoked by the tool as subprocesses
- The plugin interface covers: request interceptors, response transformers, and custom output formatters
- No plugins ship with the tool by default; the interface exists to prevent the core from absorbing every possible feature

### 7.11 Request Authentication Helpers

These are conveniences for constructing authentication-related HTTP headers and parameters. They do not involve any account system.

- Basic auth: prompt for username and password, encode as an `Authorization: Basic` header
- Bearer token: accept a token string, set as `Authorization: Bearer` header
- API key: accept a key name and value, set as a header or query parameter
- OAuth2 client credentials flow: exchange credentials for a token and use it automatically (stored locally, never sent anywhere except the configured token endpoint)

---

## 8. Non-Goals

The following are explicitly out of scope. These are not deferred features — they are rejections. Future contributors should not implement these without a fundamental reconsideration of the product's purpose.

| Item | Reason |
|---|---|
| Cloud sync | Requires a server, creates privacy risk, introduces cost |
| User accounts or sign-in | Contradicts the no-account principle entirely |
| Team collaboration features | Out of scope; Git handles collaboration |
| AI-assisted features | Adds dependencies, cost, and complexity |
| API monitoring or uptime tracking | A different product category |
| Mock server functionality | Significantly increases scope and complexity |
| API documentation hosting | A different product category |
| Built-in testing framework | Composability with existing tools is preferred |
| GUI or web UI | CLI-first; a GUI is a future consideration, not an MVP concern |
| Telemetry or analytics | Incompatible with privacy principles |
| Plugin marketplace or registry | A central registry creates infrastructure and moderation obligations |

---

## 9. CLI-First UX Principles

### 9.1 Discoverability

Every command and flag must have a `--help` output that fully describes its behavior. A new user should be able to learn the tool entirely from the terminal without consulting external documentation.

### 9.2 Composability

Output should be pipeable. A `--raw` or `--json` flag on any command should produce machine-readable output suitable for piping to `jq`, `grep`, `awk`, or any other standard tool. The tool should behave as a good Unix citizen.

### 9.3 Sensible Defaults

The most common operation — sending a GET request to a URL — should require the minimum possible input. Defaults should reflect common HTTP conventions (e.g., `Content-Type: application/json` when a JSON body is provided).

### 9.4 Explicit Over Magic

Behavior should be predictable. The tool should not silently modify requests, follow redirects without informing the user, or retry failed requests without being told to. When something is happening automatically, it should be visible.

### 9.5 Error Clarity

Error messages should state what went wrong, why it likely happened, and what the user can do to fix it. Generic error strings are not acceptable. HTTP error responses should be displayed with the same formatting as successful responses — they are not tool errors, they are API responses.

### 9.6 Exit Codes

The tool must use standard exit codes. Exit `0` on successful request execution regardless of HTTP status code (the request succeeded, even if the server returned a 404). Use non-zero exit codes for tool-level failures: connection refused, unresolvable host, malformed input. An optional `--fail-on-error` flag may exit non-zero on 4xx/5xx responses for scripting use cases.

### 9.7 No Interactivity by Default

The tool should be fully operable without interactive prompts in non-TTY environments (e.g., CI pipelines, shell scripts). Any interactive prompt must have a non-interactive equivalent via flags or environment variables.

---

## 10. Storage and Privacy Principles

### 10.1 Storage Location

All data is stored in one of two locations:

1. **Project-local storage:** A directory within or alongside the user's project, committed to version control. Collections and environment files live here.
2. **User-local storage:** A directory in the user's home directory (`~/.config/<toolname>/` or platform equivalent). Global defaults and installed plugins live here.

No data is stored in any other location. No data is ever written to a temporary directory and uploaded. No data is sent to any remote service.

### 10.2 File Formats

All persistent files use a single human-readable format (to be determined: JSON or TOML). The format choice must satisfy:

- Readable without the tool installed
- Diffable with standard Git tooling
- Writable by hand without special tooling
- Stable enough that format changes require a versioned migration

### 10.3 Secrets Handling

Environment files may contain secrets (API keys, tokens). The tool must:

- Never log secret values to stdout by default
- Provide a `--mask-secrets` mode for demo or recording scenarios
- Document clearly that secret environment files should be added to `.gitignore`
- Never transmit secret values anywhere other than the configured API endpoint

The tool does not provide a secrets manager. Users are responsible for managing their own secret storage. Integration with tools like `1Password CLI`, `direnv`, or environment variable injection is a user responsibility, not a tool responsibility.

### 10.4 No Telemetry

The tool does not collect, transmit, or store usage data, crash reports, or analytics of any kind. There is no opt-out because there is nothing to opt out of. If a future contributor proposes adding telemetry, it must be strictly opt-in, clearly disclosed, and must send data only to infrastructure the user controls.

### 10.5 Dependency Auditing

The dependency tree is a privacy surface. Each dependency must be reviewed for network activity at install or runtime. Dependencies that phone home, check for updates, or contact external services are not acceptable without explicit user consent and a clear opt-out.

---

## 11. Differentiators

The following characteristics define how this tool is positioned against existing alternatives.

| Characteristic | This Tool | Postman | Insomnia | curl / httpie |
|---|---|---|---|---|
| Account required | No | Yes (free tier) | Yes (cloud features) | No |
| Local-first storage | Yes | Partial | Partial | N/A |
| Git-friendly file format | Yes | No | Limited | N/A |
| Startup time | <100ms target | Seconds (Electron) | Seconds (Electron) | Instant |
| Open source | Yes | No | Partially | Yes |
| CLI native | Yes | No | No | Yes |
| Collections management | Yes | Yes | Yes | No |
| Environment variables | Yes | Yes | Yes | No |
| Scriptable / pipeable | Yes | Limited | No | Yes |
| Free forever | Yes | Restricted | Restricted | Yes |
| Single binary distribution | Yes | No | No | Yes |
| Telemetry | None | Yes | Yes | None |

The primary competitive claim is not feature parity with Postman. It is that this tool does one thing well — execute and manage HTTP requests from the terminal — with no overhead, no lock-in, and no compromise on privacy or simplicity.

---

## 12. Future Expansion Notes

The following items are not planned for the MVP but are not ruled out for future versions. They are listed here to prevent them from being accidentally designed around and to give future contributors a starting point for evaluation.

Any future feature must be evaluated against the core principles in Section 4 before being accepted. A feature that requires a server, requires an account, adds significant startup overhead, or makes the tool meaningfully harder to understand is out of scope regardless of user demand.

### 12.1 TUI (Terminal User Interface)

A lightweight terminal UI using a library such as Bubble Tea (Go) or Ratatui (Rust) could provide a tab-based multi-request workflow, a request history panel, and an environment switcher — all within the terminal, without a browser or Electron. This is the most likely candidate for a v2 UI surface.

A TUI must remain optional. The headless CLI interface must continue to function fully and independently.

### 12.2 gRPC Support

Adding gRPC request support would extend the tool's usefulness to backend developers working with gRPC services. This would require protobuf tooling and introduces meaningful dependency weight. It should only be considered if it can be isolated behind a flag or subcommand that does not affect the binary size or startup time of the base tool.

### 12.3 Request Chaining

The ability to pipe the output of one request as the input to another — for example, extracting a token from a login response and using it in a subsequent request — is a commonly requested feature. This can be partially addressed today via shell scripting and the tool's `--raw` output mode. A native implementation should not be added until the ergonomics of a clean, readable syntax are worked out.

### 12.4 OpenAPI / Swagger Import

Importing an OpenAPI specification and generating a local collection from it would reduce setup time for developers working with documented APIs. This is a read-only import operation with no server requirements and is broadly compatible with the tool's principles.

### 12.5 Cross-Platform GUI (Optional, Distant)

A native GUI using a framework that compiles to a small, fast binary — not Electron — could serve developers who prefer a visual interface. This must be treated as a separate application that shares the same storage format and CLI core, not as a mode of the CLI. It must not compromise the CLI experience.

### 12.6 Plugin Ecosystem

Once the plugin interface defined in the MVP is stable and validated by real usage, a curated list of community plugins (not a hosted registry) could be maintained in the project's repository. Discoverability through a simple `--list-plugins` command pointing to a static file is preferable to building any infrastructure.

---

## Appendix: Design Constraints Summary

The following constraints are load-bearing. They should be referenced in code review, architecture decisions, and contributor guidelines.

- No server-side component, ever, unless explicitly running on infrastructure the user owns and controls
- No account system of any kind
- No telemetry without explicit, local, opt-in configuration
- No Electron or browser-based runtime for the CLI
- No binary formats for user-facing storage files
- Single binary target must remain achievable on all supported platforms
- Startup time must remain under 100ms on modern hardware
- Every dependency must be justified; prefer standard library

These constraints exist not because they are easy to maintain, but because they are the product. Removing them removes the reason this tool exists.
