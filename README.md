# shade

`shade` is the command-line interface for the [Shadeform](https://www.shadeform.ai) API: deploy and manage cloud GPUs from your terminal.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=github-com/shadeform/shadeform-cli&utm_campaign=cli)
[![License: MIT](https://img.shields.io/badge/LICENSE_//_MIT-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://opensource.org/licenses/MIT)

> [!NOTE]
> `shade` is in public beta. Commands and output formats may change between minor versions; pin a release if you depend on it in scripts.

<!-- Start Summary [summary] -->
## Summary

Shadeform API: Shadeform is a single API and platform for deploying and managing cloud GPUs.
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [shade](#shade)
  * [CLI Installation](#cli-installation)
  * [Quickstart](#quickstart)
  * [Shell Completion](#shell-completion)
  * [CLI Example Usage](#cli-example-usage)
  * [For AI agents](#for-ai-agents)
  * [Authentication](#authentication)
  * [Commands](#commands)
  * [Request Body Input](#request-body-input)
  * [Server Selection](#server-selection)
  * [Output Formats](#output-formats)
  * [Error Handling](#error-handling)
  * [Diagnostics](#diagnostics)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start CLI Installation [installation] -->
## CLI Installation

### Quick Install (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/shadeform/shadeform-cli/main/scripts/install.sh | bash
```

### Quick Install (Windows PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/shadeform/shadeform-cli/main/scripts/install.ps1 | iex
```

### Go Install

Alternatively, install directly via Go:

```bash
go install github.com/shadeform/shadeform-cli/cmd/shade@latest
```

### Manual Download

Download pre-built binaries for your platform from the [releases page](https://github.com/shadeform/shadeform-cli/releases).
<!-- End CLI Installation [installation] -->

## Quickstart

Get an API key from the [Shadeform console](https://platform.shadeform.ai/settings/api), then:

```bash
# Store your API key once (OS keychain, with a config-file fallback)
shade configure

# Or export it for the current shell / CI
export SHADEFORM_API_KEY="your-api-key"

# Find the cheapest available A6000
shade instances list-types --shade-instance-type A6000 --available --sort price

# Launch one, then poll it
shade instances create --cloud hyperstack --region canada-1 --shade-instance-type A6000 --shade-cloud=true --name my-gpu
shade instances get <id>

# Fetch the private key for a Shadeform-generated SSH key (written to ~/.ssh with mode 0600)
shade ssh-keys download <ssh-key-id>

# Clean up
shade instances delete <id>
```

SSH private keys are never included in `shade ssh-keys list` or `shade ssh-keys get` output, in any format. `shade ssh-keys download` is the only command that retrieves them, and it writes to a file rather than your terminal unless you pass `--stdout`.

<!-- Start Shell Completion [completion] -->
## Shell Completion

Shell completions are available for Bash, Zsh, Fish, and PowerShell.

### Bash

```bash
# Add to ~/.bashrc:
source <(shade completion bash)

# Or install permanently:
shade completion bash > /etc/bash_completion.d/shade
```

### Zsh

```zsh
# Add to ~/.zshrc:
source <(shade completion zsh)

# Or install permanently:
shade completion zsh > "${fpath[1]}/_shade"
```

### Fish

```fish
shade completion fish | source

# Or install permanently:
shade completion fish > ~/.config/fish/completions/shade.fish
```

### PowerShell

```powershell
shade completion powershell | Out-String | Invoke-Expression
```
<!-- End Shell Completion [completion] -->

<!-- Start CLI Example Usage [usage] -->
## CLI Example Usage

### Example

```bash
shade instances list --api-key test_api_key

```
<!-- End CLI Example Usage [usage] -->

<!-- Start For AI agents [agents] -->
## For AI agents

This CLI is built to be driven by AI coding agents as well as people: everything an agent needs is discoverable from the binary itself, and every command can be validated without credentials. Work down this ladder:

| Run | You get |
|-----|---------|
| `shade --help`, `shade clusters list --help` | Commands by category, runnable examples, flags |
| `shade --usage`, `shade clusters list --usage` | The command surface as machine-readable [KDL](https://kdl.dev): commands, aliases, flags, defaults, env vars, config keys |
| `shade volumes create --schema` | The exact JSON Schema of the command's request body (all `$ref`s bundled) — build a valid `--body` from it |
| `shade clusters list --dry-run` | The exact HTTP request (method, URL, headers, body), with no credentials or network call |
| `shade clusters list --output-format json` (or `--jq`) | Machine-readable output |

### Discover the command surface

```bash
# Every command, flag, default, env var and config key, as KDL
shade --usage

# One command's subtree only
shade clusters list --usage
```

### Read the exact request schema

`--schema` is available on every command that accepts a request body (`--body`, stdin, or a whole-body flag where the command has one), including intent commands. It prints the JSON Schema the request is validated against and exits without calling the API.

```bash
# JSON Schema (draft 2020-12) of the request body, with every $ref bundled under $defs
shade volumes create --schema
```

### Probe before you spend

Start quota-spending commands with `--dry-run`. It validates inputs, resolves the request, redacts secrets and binary payloads, makes no network call, and exits 0. It never reads the OS keychain; credentials supplied by flag, environment, or config file are included only as `[REDACTED]`.

```bash
# Human preview: the [DRY-RUN] block is on stderr and stdout is empty
shade clusters list --dry-run

# Machine preview: compact JSON on stdout and silent stderr
shade clusters list --dry-run --output-format json
```

The machine form writes one object per would-be request, one per line (NDJSON for multi-request commands), with exactly this shape:

```json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
```

`body` is a parsed JSON value when the body is JSON, a string for text, `"<bytes:N>"` for binary data, and `null` when absent. An explicit caller `--jq` also selects this JSON preview protocol, but the filter is not applied to preview objects. Command-declared jq presets do not select or filter the preview.

Local mutation commands make no request under `--dry-run`: instead of a preview they emit one `{"dry_run":true,"local":true,"command":"…","message":"…"}` object. `select(.request)` keeps only would-be requests; `select(.local)` keeps the local no-ops.

### Machine-readable output

```bash
# JSON on stdout
shade clusters list --output-format json

# Filter or reshape with a jq expression (always emits JSON, overrides --output-format)
shade clusters list --jq '.'

# Print jq string results as plain text instead of JSON strings (like jq -r)
shade clusters list --jq '.' --raw-output
```

`--output-format toon` emits [TOON](https://github.com/toon-format/spec), a compact line-oriented format that uses fewer tokens than JSON; it is the default in agent mode.

### Interactive mode
Required-input prompts and guided `configure` / `auth login` forms are enabled by default. Required-input prompts require an interactive terminal; off-TTY forms read line input from stdin. Use `--no-interactive` to force flag-only execution.

```bash
# Prompt for missing command inputs
shade clusters list --interactive

# Open the guided configuration form
shade configure --interactive

# Explicitly launch the terminal command explorer
shade explore
```

### Agent mode and structured errors
Agent mode turns on automatically when a known agent environment is detected (`CLAUDECODE`, `CURSOR_AGENT`, `CODEX`, `AIDER`, `CLINE`, `WINDSURF_AGENT`, `GITHUB_COPILOT`, `AMAZON_Q`, `GEMINI_CODE_ASSIST`, `SRC_CODY`) or with `--agent-mode` (`--agent-mode=false` disables detection).
In agent mode interactive prompts never launch, output defaults to TOON, and every failure — API errors and CLI usage errors alike — is one JSON envelope on stderr:
Outside agent mode, explicit JSON and `--jq` preserve the compatibility envelope without classification; enable agent mode to request the classified contract.

```json
{
  "error": "...",
  "error_type": "validation_error",
  "error_reason": "CLI_VALIDATION",
  "exit_code": 2,
  "message": "human-readable message",
  "hints": ["what to try next"]
}
```

`error_type` is one of `authentication_error`, `authorization_error`, `not_found`, `validation_error`, `rate_limit_error`, `server_error`, `api_error`, `connection_error`, `protocol_error`, `runtime_error`, `unsupported_error`, `async_failed`, `async_timeout`, `async_unknown_state`. Classification derives from the HTTP status and transport evidence; `error_reason` is absent for API errors. Status-less local failures may use `CLI_VALIDATION`, `CLI_CONNECTION`, `CLI_PROTOCOL`, `CLI_RUNTIME`, `CLI_UNAVAILABLE`, `CLI_AUTHENTICATION`, or the async polling reasons `CLI_ASYNC_FAILED`, `CLI_ASYNC_TIMEOUT`, and `CLI_ASYNC_UNKNOWN_STATE`. `hints` preserves server guidance first, adds the most specific local taxonomy guidance, then typed CLI and command-specific guidance, removing exact duplicates. `exit_code` is always the code for the final `error_type` shown in the envelope: 1 runtime, 2 usage, or 3 authentication/authorization.
<!-- End For AI agents [agents] -->

<!-- Start Authentication [security] -->
## Authentication

Authentication credentials can be configured in four ways (in order of priority):

### 1. Command-line flags

Pass credentials directly as flags to any command:

```bash
shade --api-key "$SHADEFORM_API_KEY" clusters list
```

### 2. Environment variables

Set credentials via environment variables:

| Variable | Description |
|----------|-------------|
| `SHADEFORM_API_KEY` | API Key |

### 3. OS Keychain (recommended for workstations)

Credentials are stored securely in your operating system's keychain when you run:

```bash
shade configure
```

Secret credentials (tokens, API keys, passwords) are automatically stored in:
- **macOS**: Keychain
- **Linux**: GNOME Keyring / KWallet (via D-Bus Secret Service)
- **Windows**: Windows Credential Locker

If no keychain is available (e.g., in CI environments), credentials fall back to the config file.

### 4. Configuration file

Run the interactive `configure` command to store non-secret settings:

```bash
shade configure
```

Configuration is stored in `~/.config/shade/config.yaml`.
<!-- End Authentication [security] -->

<!-- Start Commands [operations] -->
## Commands

<details open>
<summary>Available commands</summary>

* [`instances`](docs/shade_instances.md) - Create, inspect, update, restart and delete GPU instances
  * [`list`](docs/shade_instances_list.md) - List instances
  * [`list-types`](docs/shade_instances_list-types.md) - List available instance types
  * [`create`](docs/shade_instances_create.md) - Create an instance
  * [`get`](docs/shade_instances_get.md) - Get an instance
  * [`update`](docs/shade_instances_update.md) - Update an instance
  * [`delete`](docs/shade_instances_delete.md) - Delete an instance
  * [`restart`](docs/shade_instances_restart.md) - Restart an instance
* [`clusters`](docs/shade_clusters.md) - Create, inspect and delete multi-node GPU clusters
  * [`list`](docs/shade_clusters_list.md) - List clusters
  * [`list-types`](docs/shade_clusters_list-types.md) - List available cluster types
  * [`create`](docs/shade_clusters_create.md) - Create a cluster
  * [`get`](docs/shade_clusters_get.md) - Get a cluster
  * [`delete`](docs/shade_clusters_delete.md) - Delete a cluster
* [`ssh-keys`](docs/shade_ssh-keys.md) - Manage SSH keys used to access instances
  * [`list`](docs/shade_ssh-keys_list.md) - List SSH keys
  * [`add`](docs/shade_ssh-keys_add.md) - Add an SSH key
  * [`delete`](docs/shade_ssh-keys_delete.md) - Delete an SSH key
  * [`get`](docs/shade_ssh-keys_get.md) - Get an SSH key
  * [`set-default`](docs/shade_ssh-keys_set-default.md) - Set the default SSH key
* [`volumes`](docs/shade_volumes.md) - Create, inspect and delete persistent storage volumes
  * [`list`](docs/shade_volumes_list.md) - List volumes
  * [`create`](docs/shade_volumes_create.md) - Create a volume
  * [`delete`](docs/shade_volumes_delete.md) - Delete a volume
  * [`get`](docs/shade_volumes_get.md) - Get a volume
  * [`list-types`](docs/shade_volumes_list-types.md) - List available volume types
* [`templates`](docs/shade_templates.md) - Save and reuse launch configurations as templates
  * [`list`](docs/shade_templates_list.md) - List templates
  * [`list-featured`](docs/shade_templates_list-featured.md) - List featured templates
  * [`get`](docs/shade_templates_get.md) - Get a template
  * [`save`](docs/shade_templates_save.md) - Save a template
  * [`update`](docs/shade_templates_update.md) - Update a template
  * [`delete`](docs/shade_templates_delete.md) - Delete a template

</details>
<!-- End Commands [operations] -->

<!-- Start Request Body Input [stdinpiping] -->
## Request Body Input

Commands that accept a request body take it three ways, with a clear priority chain. The examples use `shade volumes create`; every body-bearing command works the same way and prints its exact request schema with `--schema`.

### Individual flags (highest priority)

Each top-level body field is a flag:

```bash
shade volumes create --cloud 'hyperstack' --region 'canada-1' --size-in-gb 100 --name 'My storage volume'
```

### `--body` flag

Provide the entire request body as a JSON string:

```bash
shade volumes create --body '{"cloud":"hyperstack","region":"canada-1","size_in_gb":100,"name":"My storage volume"}'
```

Individual flags override `--body` values:

```bash
# Sends {"cloud":"hyperstack","region":"canada-1","size_in_gb":101,"name":"My storage volume"}
shade volumes create --body '{"cloud":"hyperstack","region":"canada-1","size_in_gb":100,"name":"My storage volume"}' --size-in-gb 101
```

### Stdin piping (lowest priority)

Pipe JSON into any command that accepts a request body:

```bash
echo '{"cloud":"hyperstack","region":"canada-1","size_in_gb":100,"name":"My storage volume"}' | shade volumes create
```

Individual flags override stdin values:

```bash
# Sends {"cloud":"hyperstack","region":"canada-1","size_in_gb":101,"name":"My storage volume"}
echo '{"cloud":"hyperstack","region":"canada-1","size_in_gb":100,"name":"My storage volume"}' | shade volumes create --size-in-gb 101
```

This is useful for chaining commands, reading from files, or scripting:

```bash
# Read body from a file
shade volumes create < request.json

# Pipe from another command
curl -s https://example.com/request.json | shade volumes create
```

### Priority

When multiple input methods are used, the priority is:

| Priority | Source | Description |
|----------|--------|-------------|
| 1 (highest) | Individual flags | `--size-in-gb ...` always wins |
| 2 | `--body` flag | Whole-body JSON via flag |
| 3 (lowest) | Stdin | Piped JSON input |
<!-- End Request Body Input [stdinpiping] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Override Server URL

Use `--server-url` to override the server URL entirely, bypassing any named or indexed server selection:

```bash
shade --server-url https://custom-api.example.com clusters list
```

**Precedence**: `--server-url` > `--server` > default
<!-- End Server Selection [server] -->

<!-- Start Output Formats [output-formats] -->
## Output Formats

Every command supports a `--output-format` flag that controls how the response is rendered to stdout.

### Available formats

| Format | Flag | Description |
|--------|------|-------------|
| Pretty | `--output-format pretty` (default) | Aligned key-value pairs with color, nested indentation. Human-readable at a glance. |
| JSON | `--output-format json` | JSON output. Passthrough when the response is already JSON (preserves original field order and numeric precision). Falls back to typed marshaling otherwise. |
| YAML | `--output-format yaml` | YAML output via standard marshaling. |
| Table | `--output-format table` | Tabular output for array responses. |
| TOON | `--output-format toon` | [Token-Oriented Object Notation](https://github.com/toon-format/spec) — a compact, line-oriented format that typically uses 30–60% fewer tokens than JSON. Well-suited for piping responses into LLM prompts. |

```bash
# Default pretty output
shade clusters list

# Machine-readable JSON
shade clusters list --output-format json

# TOON for LLM-friendly compact output
shade clusters list --output-format toon

# Pipe JSON to jq without using --output-format
shade clusters list --output-format json | jq '.'
```

### jq filtering

Use `--jq` to filter or transform the response inline using a [jq](https://jqlang.org) expression. This always outputs JSON and overrides `--output-format`:

```bash
# Extract a single field
shade clusters list --jq '.'

# Reshape with any jq program; --raw-output prints string results as plain text (like jq -r)
shade clusters list --jq '.' --raw-output
```

### Color control

Use `--color` to control terminal colors:

| Value | Behavior |
|-------|----------|
| `auto` (default) | Color when stdout is a TTY, plain text otherwise |
| `always` | Always colorize |
| `never` | Never colorize |

The `NO_COLOR` and `FORCE_COLOR` environment variables are also respected.

### Streaming and pagination

When using `--all` (pagination) or streaming operations, output is written incrementally as items arrive:

| Format | Streaming behavior |
|--------|-------------------|
| `json` | One compact JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| `yaml` | YAML documents separated by `---` |
| `toon` | One TOON-encoded object per block, separated by blank lines |
| `pretty` (default) | Pretty-printed items separated by blank lines |
<!-- End Output Formats [output-formats] -->

<!-- Start Error Handling [errors] -->
## Error Handling

The CLI uses standard exit codes to indicate success or failure:

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | Runtime/API failure |
| `2` | Usage or input failure |
| `3` | Authentication or authorization failure |

On success, the response data is printed to **stdout** as JSON. On failure, error details are printed to **stderr**.

```bash
# Capture output and handle errors
shade clusters list --output-format json > output.json 2> error.log
if [ $? -ne 0 ]; then
  echo "Error occurred, see error.log"
fi
```
This CLI uses unclassified error rendering outside agent mode: pretty and TOON print the API error text as received, while `--output-format json` and `--jq` emit the unclassified envelope (including the configure `_hint` for HTTP 401/403) plus `exit_code`. Agent mode always emits the classified JSON envelope with `exit_code`, `error_type`, optional `error_reason`, `message`, `hints`, and optional `status_code` — see [For AI agents](#for-ai-agents).
<!-- End Error Handling [errors] -->

<!-- Start Diagnostics [diagnostics] -->
## Diagnostics

The CLI includes two diagnostic flags available on all commands:

### Dry Run

Preview what would be sent without making any network calls:

```bash
shade clusters list --dry-run
```

In human output modes, stdout is empty and the `[DRY-RUN]` block goes to stderr. It includes:
- HTTP method and URL
- Request headers (sensitive values redacted)
- Request body preview (sensitive fields redacted)

With `--output-format json`, or with a caller-explicit `--jq`, stderr is silent and stdout is NDJSON: one compact preview object per would-be request. The jq filter is not applied, and command-declared jq presets do not select the JSON protocol.

```json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
```

JSON bodies remain structured; text bodies are strings; binary bodies are `"<bytes:N>"`; absent bodies are `null`. Headers retain all values as arrays, with credentials replaced by `[REDACTED]`. Dry-run never reads the OS keychain, but credentials supplied by flag, environment, or config file still appear redacted. The command exits successfully without contacting the API.

Local mutation commands emit one `{"dry_run":true,"local":true,"command":"…","message":"…"}` object in place of a preview; filter with `select(.request)` or `select(.local)`.

### Debug

Log request and response diagnostics while running normally:

```bash
shade clusters list --debug
```

Debug output goes to stderr and includes:
- Request method, URL, headers, and body preview
- Response status, headers, and body preview
- Transport errors (if any)

The command still executes normally and produces its regular output on stdout.

### Flag Precedence

If both `--dry-run` and `--debug` are set, `--dry-run` takes precedence and no network calls are made.

### Security

Sensitive information is automatically redacted in diagnostic output:
- **Headers**: `Authorization`, `Cookie`, `Set-Cookie`, `X-API-Key`, and other security headers show `[REDACTED]`
- **Body**: JSON fields named `password`, `secret`, `token`, `api_key`, `client_secret`, etc. show `[REDACTED]`
- **Binary data**: binary media and canonical base64 strings are replaced with `<bytes:N>`
- **URL query**: credential-like query parameters are replaced with `[REDACTED]`

Diagnostic output should still be treated as potentially sensitive operational data.
<!-- End Diagnostics [diagnostics] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This CLI is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

This CLI is generated programmatically. Edits to generated files are overwritten on regeneration. To customize it:

- **Configuration and behavior:** Use [OpenAPI overlays](https://www.speakeasy.com/docs/prep-openapi/overlays/create-overlays) in the Speakeasy workflow with `x-speakeasy-*` extensions (for example, `x-speakeasy-cli-commands`) to define commands, flags, help text, examples, authentication, and grouping.
- **Persistent code changes:** Store unified diffs as [patch files](https://www.speakeasy.com/docs/sdks/customize/code/patch-files/patch-files) at `.speakeasy/patches/<path-of-generated-file>.patch`; they are re-applied on every generation.

### CLI Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/shadeform/shadeform-cli&utm_campaign=cli)
