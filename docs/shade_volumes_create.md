## shade volumes create

Create a volume

### Synopsis

Create a new storage volume

```
shade volumes create [flags]
```

### Examples

```
  shade volumes create --cloud hyperstack --region canada-1 --size-in-gb 100 --name 'My storage volume'
```

### Options

```
      --body string      Request body as JSON (alternative to individual flags). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
  -c, --cloud string     Specifies the underlying cloud provider. See this [explanation](/getting-started/concepts#cloud-cloud-provider) for more details. [required]
  -h, --help             help for create
  -n, --name string      The name of the storage volume. [required]
  -r, --region string    Specifies the region. [required]
      --schema           Print the exact JSON Schema of the request body and exit
  -s, --size-in-gb int   Storage volume size in GB [required]
```

### Options inherited from parent commands

```
      --agent-mode             Enable structured errors and default TOON output for AI coding agents. Automatically enabled when a known agent environment is detected (CLAUDECODE, CURSOR_AGENT, etc.). Use --agent-mode=false to disable.
      --api-key string         API Key
      --color string           Control colored output: auto (color when output is a TTY), always, or never. Respects NO_COLOR and FORCE_COLOR env vars. (default "auto")
  -d, --debug                  Log request and response diagnostics to stderr
      --dry-run                Preview API requests without sending them (no network, no OS keychain). Human preview on stderr; with -o json or --jq, one JSON object per request on stdout. Local mutation commands (auth login, auth logout and configure) make no request: they skip prompts and writes and report a no-op (stderr, or one JSON object on stdout in the machine form)
  -H, --header stringArray     Set a custom HTTP request header (format: "Key: Value"). Can be specified multiple times.
      --include-headers        Include HTTP response headers in the output
      --interactive            Prompt for missing inputs and open guided configure/auth forms (forms fall back to line prompts on stdin off-TTY) (default true)
  -q, --jq string              Filter and transform output using a jq expression (e.g., '.name', '.items[] | .id')
      --no-interactive         Disable all interactive features (auto-prompting, explorer auto-launch, TUI forms)
  -o, --output-format string   Specify the output format. Options: pretty, json, yaml, table, toon. (default "pretty")
      --raw-output             Write --jq string results as raw text instead of JSON strings (like jq -r); non-string results stay JSON
      --server string          Select a server by index (for indexed servers) or name (for named servers)
      --server-url string      Override the default server URL
      --timeout string         HTTP request timeout (e.g., 30s, 5m, 100ms)
      --usage                  Print the CLI Usage schema in KDL format
```

### SEE ALSO

* [shade volumes](shade_volumes.md)	 - Create, inspect and delete persistent storage volumes

### Machine interface

* `shade volumes create --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade volumes create --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `shade volumes create --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
