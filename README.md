# [toolname]

A lightweight CLI API client.

Send HTTP requests, manage collections, and work with APIs directly from the terminal. No accounts, no cloud sync, no bloat.

---

## Features

- CLI-first, scriptable, and composable with standard Unix tools
- Local-first — all data stored on your machine as plain files
- YAML-based request collections, Git-friendly by design
- cURL import and export
- Environment variable support with `.env` file integration
- Cross-platform — Linux, macOS, and Windows
- Single binary, no runtime dependencies
- No accounts, no login, no telemetry
- Open-source

---

## Philosophy

This tool does one thing: send HTTP requests and manage collections of them locally. It is not a platform. It has no server component, no collaboration features, and no AI layer. It is small, fast, and intended to stay that way.

Simplicity and performance are treated as features. A request that executes in 80ms is better than one that executes in 500ms with more options. A file format that a developer can read and edit by hand is better than one that requires a GUI to be useful.

The tool is free and will remain free. There is no paid tier and no hosted infrastructure to monetize.

---

## Privacy and Local-First Design

All data — request collections, environments, configuration, and history — is stored as plain files on your machine. Nothing is sent to any server operated by this project.

- No login or account required
- No cloud sync
- No telemetry (the tool makes no outbound connections other than the requests you send)
- Files are portable, human-readable YAML that can be committed to Git

---

## Performance

The tool is written in Go and compiles to a single static binary.

- Fast startup, targeting under 100ms on modern hardware
- Minimal dependencies — the standard library is preferred throughout
- No Electron, no runtime, no interpreter

---

## Installation

**Homebrew (macOS and Linux)**

```sh
brew install [toolname]
```

**Single binary install script**

```sh
curl -fsSL https://[toolname].dev/install.sh | sh
```

**Download a binary**

Pre-built binaries for Linux, macOS, and Windows are available on the [releases page](https://github.com/[org]/[toolname]/releases).

**Build from source**

Requires Go 1.22 or later.

```sh
git clone https://github.com/[org]/[toolname].git
cd [toolname]
go build -o [toolname] ./cmd/[toolname]
```

---

## Usage

**Send a request directly**

```sh
[toolname] get https://api.example.com/users
[toolname] post https://api.example.com/users --json '{"name": "alice"}'
[toolname] get https://api.example.com/users -H "Authorization: Bearer $TOKEN"
```

**Run a saved request from a collection**

```sh
[toolname] run users.list
[toolname] run users.create --env staging
```

**Import a cURL command**

```sh
[toolname] import curl "curl -X POST https://api.example.com/users -H 'Content-Type: application/json' -d '{\"name\":\"alice\"}'"
```

**Export a saved request as cURL**

```sh
[toolname] export curl users.create
```

**Use an environment file**

```sh
[toolname] run users.list --env staging
[toolname] run users.list --var base_url=http://localhost:8080
```

**Pipe output to other tools**

```sh
[toolname] get https://api.example.com/users --raw | jq '.users[].email'
```

---

## Project Status

This project is in early development. APIs, CLI flags, and file formats may change between versions without notice.

It is not yet recommended for use in production scripts or workflows where stability is required. Feedback, bug reports, and contributions are welcome.

---

## Contributing

Contributions are welcome. Please open an issue before submitting a pull request for anything beyond small fixes, so the change can be discussed first.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on code style, commit format, and the review process.

---

## License

[MIT](LICENSE)
