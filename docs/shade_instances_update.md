## shade instances update

Update an instance

### Synopsis

Update mutable details about the instance. Set a value to null to delete it. Omit a value or leave undefined to keep unchanged.

```
shade instances update [id] [flags]
```

### Examples

```
  shade instances update --id d290f1ee-6c54-4b01-90e6-d701748f0851
```

### Options

```
      --alert string         Set a date or spend threshold to receive an email alert
      --auto-delete string   Set a date or spend threshold to automatically delete the instance
      --body string          Request body as JSON (alternative to individual flags). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
  -h, --help                 help for update
  -i, --id string            The instance id (or pass it as the [id] argument)
  -n, --name string          The name of the instance
      --schema               Print the exact JSON Schema of the request body and exit
  -t, --tags stringArray     Add custom, searchable tags to instances.
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

* [shade instances](shade_instances.md)	 - Create, inspect, update, restart and delete GPU instances

### Machine interface

* `shade instances update --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade instances update --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `shade instances update --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
