# File Format Specification

**Project:** (working title: api-testing-software)
**Version:** 0.1 — MVP
**Status:** Draft
**Last Updated:** 2026-03-28

---

## Table of Contents

1. [Overview](#1-overview)
2. [Design Goals](#2-design-goals)
3. [YAML Philosophy](#3-yaml-philosophy)
4. [Collection File Structure](#4-collection-file-structure)
5. [Request Structure](#5-request-structure)
6. [Folder and Group Organization](#6-folder-and-group-organization)
7. [Environment File Structure](#7-environment-file-structure)
8. [Variable Resolution](#8-variable-resolution)
9. [.env Integration](#9-env-integration)
10. [Global vs Project Storage](#10-global-vs-project-storage)
11. [Metadata Fields](#11-metadata-fields)
12. [Request History (Optional)](#12-request-history-optional)
13. [Portability Considerations](#13-portability-considerations)
14. [Schema Versioning Strategy](#14-schema-versioning-strategy)
15. [Example Files](#15-example-files)
16. [Future Compatibility Notes](#16-future-compatibility-notes)

---

## 1. Overview

All persistent data in this tool — request collections, environments, and configuration — is stored as plain YAML files on the local filesystem. There is no database, no binary format, no proprietary encoding, and no server-side storage.

Every file is intended to be:

- Read and edited directly in a text editor
- Committed to a Git repository alongside the code it tests
- Copied between machines with a file copy or `git clone`
- Parsed by other tools without knowledge of this tool's internals
- Understood at a glance by a developer who has never used this tool before

This specification defines the exact structure and semantics of each file type. It is the authoritative reference for both the tool's parser and anyone writing or editing files by hand.

---

## 2. Design Goals

| Goal | Description |
|---|---|
| Human-readable | Files should be understandable without documentation |
| Human-editable | Files should be writable by hand without tooling |
| Git-friendly | Changes should produce clean, meaningful diffs |
| Portable | Files must work identically on Windows, macOS, and Linux |
| Minimal | Only include structure that serves a real need |
| Stable | The format should not require frequent breaking changes |
| Tooling-friendly | Other tools should be able to parse and generate these files |
| Forward-compatible | New fields can be added without breaking existing files |

These goals are ordered by priority when they conflict. A more compact format that is harder to read loses to a slightly verbose format that is immediately clear. A clever schema that is hard to hand-edit loses to a simple one that anyone can write in 30 seconds.

---

## 3. YAML Philosophy

YAML is chosen because it supports multi-line strings naturally (critical for request bodies and GraphQL queries), allows inline comments, is familiar to developers who work with configuration files, and produces readable diffs.

The following YAML features are used in this format:

- Scalar strings (quoted and unquoted)
- Block scalars (`|` for literal, `>` for folded) for multi-line bodies
- Mappings (key-value objects)
- Sequences (lists)
- Comments (`#`)

The following YAML features are intentionally avoided:

- Anchors and aliases (`&`, `*`) — they obscure what a file actually contains
- Complex types and tags (`!!python/object`, etc.)
- Implicit type coercion (e.g., unquoted `true`, `null`, `1.0` where a string is intended)
- Multi-document streams (`---` separators within a single file, except for the optional schema version header)

### Quoting Strings

URLs and header values must always be quoted to prevent YAML from misinterpreting special characters (`:`, `{`, `}`, `#`, `*`).

Variable references (`${variable}`) must also be quoted when they appear at the start of a value or inside a URL, as the `$` and `{` characters can confuse some YAML parsers.

```yaml
# Correct
url: "https://api.example.com/users/${user_id}"
headers:
  Authorization: "Bearer ${auth_token}"

# Incorrect — YAML may misparse the colon and braces
url: https://api.example.com/users/${user_id}
```

When in doubt, quote the value. Unnecessary quotes never break parsing; missing quotes sometimes do.

---

## 4. Collection File Structure

A collection is a single YAML file containing a named group of saved requests. Collections map naturally to the projects or APIs they test — one collection per service, or one collection per functional area.

### 4.1 Top-Level Structure

```yaml
# Schema version is optional. Omit if not needed.
# When present, it must be the first field.
schema_version: 1

name: User API
description: Requests for the user management service
base_url: "https://api.example.com"

# Optional metadata
tags: [users, auth, internal]
created_at: "2026-03-28"
updated_at: "2026-03-28"

# Requests at the top level (flat collection)
requests:
  - name: List Users
    method: GET
    url: "${base_url}/users"

  - name: Get User
    method: GET
    url: "${base_url}/users/${user_id}"
```

### 4.2 Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `schema_version` | integer | No | Format version. Omit if not needed. See Section 14. |
| `name` | string | Yes | Human-readable collection name |
| `description` | string | No | What this collection is for |
| `base_url` | string | No | Default base URL for all requests in this collection |
| `tags` | list of strings | No | Arbitrary labels for filtering and organization |
| `created_at` | string (ISO 8601) | No | Creation date |
| `updated_at` | string (ISO 8601) | No | Last modified date |
| `requests` | list | No | Flat list of requests (see Section 5) |
| `folders` | list | No | Nested groups of requests (see Section 6) |

A collection file may contain `requests`, `folders`, or both. An empty collection with neither is valid.

---

## 5. Request Structure

A request is the fundamental unit of the file format. Every request is a self-contained description of one HTTP call.

### 5.1 Full Request Schema

```yaml
- name: Create User
  description: Create a new user account

  method: POST
  url: "${base_url}/users"

  # Request headers
  headers:
    Authorization: "Bearer ${auth_token}"
    Content-Type: application/json
    X-Request-Source: apitool

  # Query parameters (appended to URL)
  params:
    format: json
    version: "2"

  # Request body — use block scalar for multi-line JSON or text
  body: |
    {
      "name": "${user_name}",
      "email": "${user_email}",
      "role": "member"
    }

  # Optional metadata
  tags: [create, users]
  created_at: "2026-03-28"
  updated_at: "2026-03-28"
```

### 5.2 Request Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Human-readable request name. Used to reference the request from the CLI. |
| `description` | string | No | What this request does |
| `method` | string | Yes | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS` |
| `url` | string | Yes | Full URL or path with variable references |
| `headers` | mapping | No | Request headers as key-value pairs |
| `params` | mapping | No | Query parameters as key-value pairs |
| `body` | string | No | Raw request body. Use block scalar (`|`) for multi-line content. |
| `tags` | list of strings | No | Arbitrary labels |
| `created_at` | string (ISO 8601) | No | Creation date |
| `updated_at` | string (ISO 8601) | No | Last modified date |

### 5.3 HTTP Method Casing

The `method` field is case-insensitive at parse time but should be written in uppercase by convention. The tool normalizes it to uppercase before sending.

### 5.4 Body Format

The `body` field is always a raw string. The tool does not interpret the body format — it sends the value as-is. Setting the correct `Content-Type` header is the user's responsibility.

For JSON bodies, use a YAML block scalar to preserve readability:

```yaml
body: |
  {
    "name": "Alice",
    "role": "admin"
  }
```

For form-encoded bodies, write the value as a single string:

```yaml
headers:
  Content-Type: application/x-www-form-urlencoded
body: "name=Alice&role=admin"
```

### 5.5 Request Naming and Reference

Request names are free-form strings. Within a CLI invocation, requests are referenced by their name as it appears in the YAML file. Names are case-sensitive. Names within the same collection file should be unique; the tool will warn on duplicates.

Names are not IDs. They can be changed without breaking anything except shell scripts or aliases that reference them by name. A future version of the format may introduce optional stable IDs alongside names; see Section 16.

---

## 6. Folder and Group Organization

Folders allow requests within a single collection file to be grouped logically. Folders may be nested to any depth, though deeply nested structures should be avoided in favor of splitting into multiple collection files.

### 6.1 Folder Structure

```yaml
name: User API
base_url: "https://api.example.com"

folders:
  - name: Users
    description: User CRUD operations
    requests:
      - name: List Users
        method: GET
        url: "${base_url}/users"

      - name: Get User
        method: GET
        url: "${base_url}/users/${user_id}"

      - name: Create User
        method: POST
        url: "${base_url}/users"
        headers:
          Content-Type: application/json
        body: |
          {
            "name": "${user_name}",
            "email": "${user_email}"
          }

  - name: Authentication
    description: Login and token management
    requests:
      - name: Login
        method: POST
        url: "${base_url}/auth/login"
        headers:
          Content-Type: application/json
        body: |
          {
            "email": "${user_email}",
            "password": "${user_password}"
          }

      - name: Refresh Token
        method: POST
        url: "${base_url}/auth/refresh"
        headers:
          Authorization: "Bearer ${refresh_token}"
```

### 6.2 Nested Folders

Folders may contain both `requests` and nested `folders`:

```yaml
folders:
  - name: Admin
    folders:
      - name: Users
        requests:
          - name: List All Users
            method: GET
            url: "${base_url}/admin/users"
      - name: Settings
        requests:
          - name: Get System Config
            method: GET
            url: "${base_url}/admin/config"
    requests:
      - name: Admin Health Check
        method: GET
        url: "${base_url}/admin/health"
```

### 6.3 Folder Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Human-readable folder name |
| `description` | string | No | What requests in this folder are for |
| `requests` | list | No | Requests directly in this folder |
| `folders` | list | No | Nested sub-folders |
| `tags` | list of strings | No | Labels applied to all requests in this folder |

### 6.4 Referencing Requests Inside Folders

When using the CLI to run a named request inside a folder, the tool uses a dot-separated path that mirrors the folder hierarchy:

```sh
# Run "List Users" inside the "Users" folder
apitool run "Users.List Users"

# Run "List All Users" inside Admin > Users
apitool run "Admin.Users.List All Users"
```

Folder path separators are dots. Spaces within a name component are preserved. When a name contains a dot, it must be escaped with a backslash: `Admin\.V2.Users.List`.

---

## 7. Environment File Structure

Environments hold named variable values that are substituted into requests at runtime. An environment might represent a deployment target (local, staging, production) or a persona (admin user, regular user).

Two environment file formats are supported and may coexist.

### 7.1 Single-Environment File

One environment per file. The filename determines the environment name.

```yaml
# envs/staging.yaml
# Environment name is derived from the filename: "staging"

base_url: "https://staging.api.example.com"
api_version: v2
request_timeout: 30

# Sensitive values should be left empty here and overridden
# via a .env file or environment variable. See Section 9.
auth_token: ""
user_password: ""
```

### 7.2 Multi-Environment File

Multiple named environments in a single file. Useful for simple projects where a single file is easier to manage than several.

```yaml
# envs/environments.yaml

environments:
  local:
    base_url: "http://localhost:8080"
    api_version: v1
    request_timeout: 10
    auth_token: ""

  staging:
    base_url: "https://staging.api.example.com"
    api_version: v2
    request_timeout: 30
    auth_token: ""

  production:
    base_url: "https://api.example.com"
    api_version: v2
    request_timeout: 60
    auth_token: ""
```

### 7.3 Format Detection

The tool distinguishes between the two formats by the presence of the `environments` top-level key:

- If the file contains an `environments` key: treat it as a multi-environment file
- Otherwise: treat it as a single-environment file

Both formats are valid. Both can be committed to the same repository. The single-environment format is preferred when environments have meaningfully different content. The multi-environment format is preferred for simple projects where all environments share the same variable names.

### 7.4 Environment Field Reference

Individual environment variables are flat key-value pairs. Values are always strings at the YAML level, even when they represent numbers. Variable values are substituted as raw strings during interpolation; no type conversion occurs.

| Field | Notes |
|---|---|
| Any string key | Variable name, used as `${key}` in requests |
| Values | Always treated as strings |
| Nested objects | Not supported in v1. All variables must be top-level keys. |

---

## 8. Variable Resolution

Variable references in request files are written as `${variable_name}` and resolved at runtime before the request is sent.

### 8.1 Syntax

```yaml
url: "${base_url}/users/${user_id}"
headers:
  Authorization: "Bearer ${auth_token}"
body: |
  {
    "email": "${user_email}"
  }
```

Variable references may appear in: `url`, `headers` values, `params` values, and `body`. They may not appear in field names (keys).

### 8.2 Resolution Order

Variables are resolved using the following precedence, from highest to lowest:

```
1. CLI flag overrides     (--var key=value)
2. OS environment variables
3. .env file variables    (.env.local → .env → .env.<environment>)
4. Active environment file variables
5. Collection-level base_url (special case, see below)
```

A variable defined at a higher level always wins. This allows a base value to be set in a committed environment file and overridden locally by a `.env.local` file or a CLI flag without modifying the committed file.

### 8.3 The `base_url` Special Case

The `base_url` field at the collection root is syntactic sugar. It is injected into the variable context under the key `base_url` before environment variables are applied. If an environment file also defines `base_url`, the environment file value takes precedence and the collection-level value is ignored. This allows per-environment URL overrides without modifying the collection file.

### 8.4 Undefined Variables

If a variable reference in a request cannot be resolved, the tool:

1. Emits a warning to stderr naming the unresolved variable
2. Sends the request with the literal `${variable_name}` string in place of the value

The tool does not fail silently or substitute an empty string. The literal passthrough makes it immediately obvious in the response that a variable was not set.

### 8.5 Variable Naming

Variable names may contain letters, digits, and underscores. They are case-sensitive. By convention, variable names use `snake_case`.

```
${auth_token}     ✓ valid
${AUTH_TOKEN}     ✓ valid (conventional for OS env vars)
${base-url}       ✗ invalid (hyphens not allowed)
${baseUrl}        ✓ valid (but snake_case preferred)
```

---

## 9. .env Integration

Secrets and environment-specific values that should not be committed to version control are stored in `.env`-style files. These are plain text files with `KEY=VALUE` pairs, one per line.

### 9.1 Supported .env Files

The tool looks for `.env` files in the following locations, in order from lowest to highest priority:

```
.env                    Base values, may be committed (no secrets)
.env.<environment>      Environment-specific values (e.g., .env.staging)
.env.local              Local overrides, always gitignored, highest priority
```

All three files are loaded if they exist. Values from higher-priority files override lower-priority files.

### 9.2 .env File Format

```sh
# .env.local — local secrets, never committed

AUTH_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
USER_PASSWORD=hunter2
DB_URL=postgres://localhost:5432/myapp_dev
```

Rules:
- One variable per line: `KEY=VALUE`
- Lines beginning with `#` are comments and are ignored
- Blank lines are ignored
- Keys are conventionally `UPPER_SNAKE_CASE` but this is not enforced
- Values are taken as literal strings from the `=` to the end of the line
- Values may optionally be quoted with single or double quotes; quotes are stripped
- No variable interpolation within `.env` files (values are always literal)

### 9.3 .env Variable Lookup

When the tool resolves a variable reference like `${auth_token}`, it checks the `.env` context using a case-insensitive key match. Both `auth_token` and `AUTH_TOKEN` in a `.env` file will satisfy a `${auth_token}` reference.

This case-insensitive bridge means a YAML environment file can use `snake_case` keys consistently while OS environment variables and `.env` files use `UPPER_SNAKE_CASE` — the common convention for secrets.

### 9.4 Recommended .gitignore Entries

Projects using this tool should include the following in their `.gitignore`:

```
# API tool secrets
.env.local
.env.*.local
.toolname/envs/local.yaml
```

The base `.env` file and named environment files (`.env.staging`, `.env.production`) without secrets may be committed to document which variables are expected without revealing their values:

```sh
# .env — committed, documents expected variables

AUTH_TOKEN=
USER_PASSWORD=
API_KEY=
```

An empty value signals to other developers that the variable must be set locally, without exposing the actual value.

---

## 10. Global vs Project Storage

The tool uses two storage locations. The resolution rules are consistent and predictable.

### 10.1 Directory Locations

**Global storage** — for user-personal collections and environments not tied to any project:

| Platform | Path |
|---|---|
| Linux | `~/.config/apitool/` |
| macOS | `~/Library/Application Support/apitool/` |
| Windows | `%APPDATA%\apitool\` |

**Project-local storage** — for collections and environments that belong to a project and should live in its repository:

```
<project-root>/
  .apitool/
    collections/
    envs/
    config.yaml
```

The project root is determined by walking up the directory tree from the current working directory until a `.apitool/` directory is found — the same discovery mechanism used by Git.

### 10.2 Full Directory Layout

```
# Global
~/.config/apitool/
  config.yaml
  collections/
    personal.yaml
    scratch.yaml
  envs/
    default.yaml
    home-lab.yaml

# Project-local (committed to the repository)
<project-root>/
  .apitool/
    config.yaml           # project config overrides
    collections/
      users-api.yaml
      payments-api.yaml
    envs/
      shared.yaml         # committed: non-secret base variables
      local.yaml          # gitignored: secrets and local overrides
```

### 10.3 Resolution Rules

**Collections:** When a named collection is requested, the tool checks the project-local collections directory first, then the global collections directory. The first match wins.

**Environments:** When a named environment is requested, the tool checks the project-local envs directory first, then the global envs directory. The first match wins. This allows a project to define a `staging` environment that shadows the user's global `staging` environment.

**Config:** The project config overlays the global config. Keys present in the project config replace the corresponding global config key. Keys absent from the project config inherit the global value.

### 10.4 Explicit Paths

The `--collection` and `--env` flags accept explicit file paths, bypassing the discovery mechanism entirely. This is useful for scripts that need deterministic behavior regardless of the current working directory:

```sh
apitool run "List Users" \
  --collection ./tests/api/users.yaml \
  --env ./tests/envs/staging.yaml
```

---

## 11. Metadata Fields

All file types support optional metadata fields. Metadata is never required for the tool to function — it exists to make files more self-documenting and easier to manage at scale.

### 11.1 Supported Metadata Fields

| Field | Type | Applies To | Description |
|---|---|---|---|
| `description` | string | Collection, Folder, Request | Human-readable explanation of purpose |
| `tags` | list of strings | Collection, Folder, Request | Labels for filtering and organization |
| `created_at` | string (ISO 8601 date) | Collection, Request | When the item was first created |
| `updated_at` | string (ISO 8601 date) | Collection, Request | When the item was last modified |

### 11.2 Date Format

Dates use ISO 8601 format: `YYYY-MM-DD`. Full timestamps (`2026-03-28T14:30:00Z`) are also accepted but the date-only form is preferred for readability and to avoid timezone noise in Git diffs.

### 11.3 Tags

Tags are free-form strings with no enforced vocabulary. They are intended for human use (filtering, searching, organizing) rather than machine use. A future CLI version may support `--tag` filtering on `collection list` output.

```yaml
tags: [auth, critical, v2-only]
```

### 11.4 Metadata Does Not Affect Execution

Metadata fields are read by the tool but never affect how a request is sent. Adding, changing, or removing metadata does not change request behavior. This is an invariant.

---

## 12. Request History (Optional)

Request history is disabled by default. When enabled, the tool records a log of executed requests and their responses as local files.

### 12.1 Enabling History

History is enabled in the config file:

```yaml
# config.yaml
history:
  enabled: true
  max_entries: 100          # Maximum entries to retain; oldest are pruned
  store_response_body: true # Set to false to store metadata only
```

### 12.2 History Storage Location

History is always stored in the global config directory and is never stored in a project's repository:

```
~/.config/apitool/
  history/
    2026-03-28T14-30-00Z.yaml
    2026-03-28T14-32-15Z.yaml
```

History files are named by timestamp. They are never committed to Git.

### 12.3 History Entry Format

```yaml
# history/2026-03-28T14-30-00Z.yaml

timestamp: "2026-03-28T14:30:00Z"
collection: users-api
request_name: Create User
environment: staging

request:
  method: POST
  url: "https://staging.api.example.com/users"
  headers:
    Authorization: "Bearer [redacted]"
    Content-Type: application/json
  body: |
    {
      "name": "Alice",
      "email": "alice@example.com"
    }

response:
  status: 201
  status_text: Created
  duration_ms: 143
  size_bytes: 312
  headers:
    Content-Type: application/json
    X-Request-Id: abc-123
  body: |
    {
      "id": "usr_9f3k2",
      "name": "Alice",
      "email": "alice@example.com",
      "created_at": "2026-03-28T14:30:00Z"
    }
```

### 12.4 Secret Redaction

When history is enabled, any header value for keys matching a known sensitive pattern (`Authorization`, `Cookie`, `X-Api-Key`, and others) is replaced with `[redacted]` before writing to disk. Request body content is stored as-is; users should be aware that secrets embedded in request bodies will be persisted to the history file.

The list of redacted header names is configurable:

```yaml
history:
  enabled: true
  redact_headers:
    - Authorization
    - Cookie
    - X-Api-Key
    - X-Auth-Token
```

---

## 13. Portability Considerations

### 13.1 No Machine-Specific Paths

Collection files, environment files, and configuration files must never contain absolute filesystem paths. All file references use relative paths or variable references. A file that works on one machine must work identically on another after cloning the repository.

### 13.2 No Binary Content

All files are plain UTF-8 text. Binary content (images, certificates, binary request bodies) is not supported as inline YAML. If a request requires a binary body, this is out of scope for v1.

### 13.3 Line Endings

All files written by the tool use Unix line endings (`\n`). On Windows, Git's `autocrlf` setting handles normalization. The tool reads files with either line ending without error.

The repository should include a `.gitattributes` file to ensure consistent line endings:

```
# .gitattributes
*.yaml  text eol=lf
*.yml   text eol=lf
.env*   text eol=lf
```

### 13.4 Encoding

All YAML files are UTF-8. Non-ASCII characters in names, descriptions, and body content are fully supported. The `charset` declaration is not required.

### 13.5 File Permissions

On Unix systems, the tool creates files with mode `0644` (readable by others, writable by owner). Files that contain secrets (`.env.local`, `local.yaml`) should be tightened by the user to `0600`, though the tool does not enforce this in v1.

### 13.6 Interoperability

The YAML schema used by this tool is intentionally simple. Any YAML parser can read these files. Any developer familiar with YAML can understand them. The format does not use custom tags, custom directives, or any YAML feature beyond the core subset described in Section 3.

Other tools wishing to import or export this format need only implement a basic YAML parser and the schema described in this document. There is no SDK, no proprietary format, and no hidden state.

---

## 14. Schema Versioning Strategy

### 14.1 Version Field

All file types accept an optional `schema_version` integer field as the first key in the file. In the current version of the format, the value is `1`.

```yaml
schema_version: 1

name: User API
# ...
```

Omitting `schema_version` is equivalent to setting it to `1`. The field is not required.

### 14.2 When Versioning Matters

The schema version is incremented only for breaking changes — changes that would cause an existing valid file to be parsed incorrectly or rejected. Additive changes (new optional fields) never require a version increment.

The following are breaking changes that require a version increment:

- Removing a field the parser currently reads
- Changing the meaning or type of an existing field
- Making a previously optional field required
- Changing the variable syntax from `${var}` to another form

The following are non-breaking changes that do not require a version increment:

- Adding a new optional field
- Relaxing a validation rule
- Adding a new allowed value for an enum field

### 14.3 Version Handling

When the tool reads a file with a `schema_version` higher than its own supported maximum, it emits a warning and attempts to parse the file with the current schema. This allows files written by a newer version of the tool to be partially usable by an older version.

When the tool reads a file with a `schema_version` it explicitly knows is incompatible, it emits an error describing the incompatibility and the minimum tool version required.

### 14.4 Safe Introduction

Because `schema_version` is optional and defaults to `1`, introducing this field into an existing file is always a safe, backward-compatible change. Files without the field continue to parse correctly.

---

## 15. Example Files

### 15.1 Simple Flat Collection

```yaml
# .apitool/collections/users-api.yaml
# A flat list of requests for the Users API.

schema_version: 1

name: Users API
description: CRUD operations for the user management service
base_url: "${base_url}"
tags: [users, rest]
updated_at: "2026-03-28"

requests:
  - name: List Users
    description: Returns a paginated list of all users
    method: GET
    url: "${base_url}/users"
    headers:
      Authorization: "Bearer ${auth_token}"
    params:
      page: "1"
      limit: "25"

  - name: Get User
    description: Returns a single user by ID
    method: GET
    url: "${base_url}/users/${user_id}"
    headers:
      Authorization: "Bearer ${auth_token}"

  - name: Create User
    method: POST
    url: "${base_url}/users"
    headers:
      Authorization: "Bearer ${auth_token}"
      Content-Type: application/json
    body: |
      {
        "name": "${user_name}",
        "email": "${user_email}",
        "role": "member"
      }

  - name: Update User
    method: PATCH
    url: "${base_url}/users/${user_id}"
    headers:
      Authorization: "Bearer ${auth_token}"
      Content-Type: application/json
    body: |
      {
        "name": "${user_name}"
      }

  - name: Delete User
    method: DELETE
    url: "${base_url}/users/${user_id}"
    headers:
      Authorization: "Bearer ${auth_token}"
```

---

### 15.2 Nested Collection with Folders

```yaml
# .apitool/collections/platform-api.yaml
# Requests organized into functional groups.

schema_version: 1

name: Platform API
description: Full platform API covering auth, users, and billing
base_url: "${base_url}"
updated_at: "2026-03-28"

folders:
  - name: Authentication
    description: Login, logout, and token operations
    requests:
      - name: Login
        method: POST
        url: "${base_url}/auth/login"
        headers:
          Content-Type: application/json
        body: |
          {
            "email": "${user_email}",
            "password": "${user_password}"
          }

      - name: Logout
        method: POST
        url: "${base_url}/auth/logout"
        headers:
          Authorization: "Bearer ${auth_token}"

      - name: Refresh Token
        method: POST
        url: "${base_url}/auth/refresh"
        headers:
          Content-Type: application/json
        body: |
          {
            "refresh_token": "${refresh_token}"
          }

  - name: Users
    description: User account management
    folders:
      - name: Admin
        description: Admin-only user operations
        requests:
          - name: List All Users
            method: GET
            url: "${base_url}/admin/users"
            headers:
              Authorization: "Bearer ${auth_token}"
            params:
              include_deleted: "true"

          - name: Impersonate User
            method: POST
            url: "${base_url}/admin/users/${user_id}/impersonate"
            headers:
              Authorization: "Bearer ${auth_token}"

    requests:
      - name: Get Current User
        method: GET
        url: "${base_url}/users/me"
        headers:
          Authorization: "Bearer ${auth_token}"

      - name: Update Current User
        method: PATCH
        url: "${base_url}/users/me"
        headers:
          Authorization: "Bearer ${auth_token}"
          Content-Type: application/json
        body: |
          {
            "name": "${user_name}"
          }

  - name: Billing
    description: Subscription and payment operations
    tags: [billing, stripe]
    requests:
      - name: Get Subscription
        method: GET
        url: "${base_url}/billing/subscription"
        headers:
          Authorization: "Bearer ${auth_token}"

      - name: Cancel Subscription
        method: DELETE
        url: "${base_url}/billing/subscription"
        headers:
          Authorization: "Bearer ${auth_token}"
```

---

### 15.3 Single-Environment File

```yaml
# .apitool/envs/staging.yaml
# Staging environment — commit this file.
# Set auth_token and user_password in .env.local, not here.

base_url: "https://staging.api.example.com"
api_version: v2
request_timeout: "30"

# Variable names for secrets — values left empty intentionally.
# Override these in .env.local or via OS environment variables.
auth_token: ""
refresh_token: ""
user_email: "testuser@example.com"
user_password: ""
user_id: "usr_test001"
user_name: "Test User"
```

---

### 15.4 Multi-Environment File

```yaml
# .apitool/envs/environments.yaml
# All environments in one file.
# Commit this file. Override secrets in .env.local.

environments:
  local:
    base_url: "http://localhost:8080"
    api_version: v1
    request_timeout: "10"
    auth_token: ""
    user_email: "dev@localhost"
    user_password: ""
    user_id: "usr_dev001"
    user_name: "Local Dev"

  staging:
    base_url: "https://staging.api.example.com"
    api_version: v2
    request_timeout: "30"
    auth_token: ""
    user_email: "qa@example.com"
    user_password: ""
    user_id: "usr_qa001"
    user_name: "QA User"

  production:
    base_url: "https://api.example.com"
    api_version: v2
    request_timeout: "60"
    auth_token: ""
    user_email: ""
    user_password: ""
    user_id: ""
    user_name: ""
```

---

### 15.5 .env File Usage

Commit a `.env` that documents expected variables without values:

```sh
# .env — committed to the repository
# Copy this to .env.local and fill in values for local development.

AUTH_TOKEN=
REFRESH_TOKEN=
USER_PASSWORD=
```

Keep actual secrets in `.env.local`, which is gitignored:

```sh
# .env.local — never committed
# Local development secrets.

AUTH_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c3JfZGV2MDAxIn0.signature
REFRESH_TOKEN=dGhpcyBpcyBub3QgYSByZWFsIHRva2Vu
USER_PASSWORD=hunter2
```

Reference these in request files using the lowercase variable syntax:

```yaml
headers:
  Authorization: "Bearer ${auth_token}"
```

The tool resolves `${auth_token}` by checking (in order): CLI flags, OS environment, `.env.local`, `.env`, then the active environment YAML file. The `AUTH_TOKEN` value from `.env.local` satisfies the `${auth_token}` reference via the case-insensitive bridge described in Section 9.3.

---

### 15.6 Variable Usage in a Request

```yaml
# Demonstrates all locations where variables may appear.

- name: Search Users
  method: GET

  # Variables in the URL path and base
  url: "${base_url}/api/${api_version}/users"

  headers:
    # Variable in a header value
    Authorization: "Bearer ${auth_token}"
    # Literal value — no variable needed
    Accept: application/json

  params:
    # Variable in a query parameter
    q: "${search_query}"
    # Literal query parameter
    format: json

  # Variables inside a JSON body (body is a raw string, not parsed YAML)
  body: |
    {
      "filter": {
        "name": "${user_name}",
        "role": "${user_role}"
      },
      "page": 1
    }
```

---

## 16. Future Compatibility Notes

This section describes known areas where the format may need to evolve and how those evolutions will be handled without breaking existing files.

### 16.1 Stable Request IDs

The current format uses human-readable names as the only identifier for requests. This is intentional for v1 — names are readable in Git history and in the CLI. A future version may add an optional `id` field containing a stable, tool-generated identifier (likely a short UUID or content hash).

```yaml
# Future: optional stable ID alongside name
- id: req_7f2a1b
  name: List Users
  method: GET
  url: "${base_url}/users"
```

Because `id` would be optional, adding it does not break any existing file. The tool will never require an ID to execute a request.

### 16.2 Request-Level Authentication Helpers

A future version may add a convenience `auth` block to requests to reduce boilerplate for common authentication patterns:

```yaml
# Future: optional auth block (not in v1)
- name: List Users
  method: GET
  url: "${base_url}/users"
  auth:
    type: bearer
    token: "${auth_token}"
```

The `auth` block, if added, will be syntactic sugar that expands to the equivalent `Authorization` header at parse time. It will be entirely optional — existing requests using explicit `Authorization` headers continue to work without modification.

This `auth` block refers exclusively to request-level HTTP authentication. It has no relationship to user accounts, login, or any form of identity within the tool itself.

### 16.3 GraphQL Request Type

GraphQL support, when added, will extend the request schema with a `graphql` block:

```yaml
# Future: GraphQL request type (not in v1)
- name: Get User
  method: POST
  url: "${base_url}/graphql"
  graphql:
    query: |
      query GetUser($id: ID!) {
        user(id: $id) {
          id
          name
          email
        }
      }
    variables:
      id: "${user_id}"
```

The `graphql` block will coexist with the existing `body` field. When `graphql` is present, the tool constructs the JSON body automatically. The `body` field, if also present, is ignored.

### 16.4 Script Hooks

A future version may add optional pre/post script hooks to requests or collections, enabling lightweight automation without a full testing framework:

```yaml
# Future: script hooks (not in v1)
- name: Create User
  method: POST
  url: "${base_url}/users"
  hooks:
    post: |
      export USER_ID=$(echo $RESPONSE_BODY | jq -r '.id')
```

If added, hooks will be shell command strings executed as subprocesses with response data injected as environment variables. They will be optional, explicitly named (`hooks:`), and will not affect requests that omit them.

### 16.5 Format Evolution Commitment

The file format described in this document will not change in a breaking way without:

1. A `schema_version` increment
2. A documented migration path
3. A deprecation period during which the old format continues to be supported

The goal is that a collection file written today should still parse and execute correctly with any future version of the tool, with no modification required.
