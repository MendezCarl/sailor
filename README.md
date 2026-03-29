# Sailor

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

Sailor does one thing: send HTTP requests and manage collections of them locally. It is not a platform. It has no server component, no collaboration features, and no AI layer. It is small, fast, and intended to stay that way.

Simplicity and performance are treated as features. A request that executes in 80ms is better than one that executes in 500ms with more options. A file format that a developer can read and edit by hand is better than one that requires a GUI to be useful.

Sailor is free and will remain free. There is no paid tier and no hosted infrastructure to monetize.

---

## Privacy and Local-First Design

All data — request collections, environments, configuration, and history — is stored as plain files on your machine. Nothing is sent to any server operated by this project.

- No login or account required
- No cloud sync
- No telemetry (the tool makes no outbound connections other than the requests you send)
- Files are portable, human-readable YAML that can be committed to Git

---

## Performance

Sailor is written in Go and compiles to a single static binary.

- Fast startup, targeting under 100ms on modern hardware
- Minimal dependencies — the standard library is preferred throughout
- No Electron, no runtime, no interpreter

---

## Installation

**Homebrew (macOS and Linux)**

```sh
brew install sailor
```

**Download a binary**

Pre-built binaries for Linux, macOS, and Windows are available on the [releases page](https://github.com/MendezCarl/sailor/releases). Download the archive for your platform, extract it, and place `sailor` somewhere on your `$PATH`.

**Build from source**

Requires Go 1.22 or later.

```sh
git clone https://github.com/MendezCarl/sailor.git
cd sailor
make build
```

---

## Usage

**Send a request from a file**

```sh
sailor send -f examples/demo.yaml
sailor send -f examples/post-json.yaml --var base_url=https://api.example.com
```

**Run a named request from a collection**

```sh
sailor run "List Posts" --collection examples/posts-collection.yaml
sailor run "Get Post" --collection examples/posts-collection.yaml --headers
sailor run "Create Post" --collection examples/posts-collection.yaml --var base_url=http://localhost:8080
```

**Import a cURL command**

```sh
sailor import curl 'curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d "{\"name\":\"alice\"}"'
sailor import curl 'curl https://api.example.com/users' --output request.yaml
```

**Export a saved request as cURL**

```sh
sailor export curl -f examples/demo.yaml
sailor export curl --collection examples/posts-collection.yaml "Get Post"
```

**Use environment variables**

```sh
# Set variables in an .env file
echo "BASE_URL=https://api.example.com" > .env
sailor send -f examples/demo.yaml

# Or pass them directly
sailor send -f examples/demo.yaml --var base_url=http://localhost:8080
```

**Pipe output to other tools**

```sh
sailor send -f examples/demo.yaml --raw | jq '.title'
sailor run "List Posts" --collection examples/posts-collection.yaml --raw | jq '.[0]'
```

**Output modes**

```sh
sailor send -f examples/demo.yaml --json | jq '.status_code'   # structured JSON output
sailor send -f examples/demo.yaml --raw                         # body only, no decoration
sailor send -f examples/demo.yaml --quiet                       # suppress status line
sailor send -f examples/demo.yaml --headers                     # show response headers
```

**Scripting**

```sh
sailor send -f examples/demo.yaml --fail-on-error && echo "ok"
sailor send -f examples/demo.yaml --timeout 5s
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
