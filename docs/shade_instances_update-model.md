## shade instances update-model

Update an instance's model configuration

### Synopsis

Queues a new model configuration for an instance launched to serve a model. The agent on the instance applies it, so a 200 means the change was accepted and returns the id of the queued entry. Only instances launched with a model configuration accept this.

```
shade instances update-model [instance-id] [flags]
```

### Examples

```
  shade instances update-model --instance-id <id> --model-api-key <value>
```

### Options

```
      --body string               Request body as JSON (alternative to individual flags). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
  -h, --help                      help for update-model
  -i, --instance-id string        The ID of the instance, from the id field of GET /v1/instances or the create response. (or pass it as the [instance-id] argument)
      --model-api-key string      The API key the served model's inference endpoint will require from callers. Not your Shadeform API key. [required]
      --model-properties string   The model to load and how to serve it.
      --schema                    Print the exact JSON Schema of the request body and exit
```

### Options inherited from parent commands

```
      --agent-mode             Enable structured errors and default TOON output for AI coding agents. Automatically enabled when a known agent environment is detected (CLAUDECODE, CURSOR_AGENT, etc.). Use --agent-mode=false to disable.
      --api-key string         API key issued from the Shadeform dashboard (Settings → API Keys).
      --color string           Control colored output: auto (color when output is a TTY), always, or never. Respects NO_COLOR and FORCE_COLOR env vars. (default "auto")
  -d, --debug                  Log request and response diagnostics to stderr
      --dry-run                Preview API requests without sending them (no network, no OS keychain). Human preview on stderr; with -o json or --jq, one JSON object per request on stdout. Local mutation commands (auth login, auth logout and configure) make no request: they skip prompts and writes and report a no-op (stderr, or one JSON object on stdout in the machine form)
  -H, --header stringArray     Set a custom HTTP request header (format: "Key: Value"). Can be specified multiple times.
      --include-headers        Include HTTP response headers in the output
      --interactive            Prompt for missing inputs and open guided configure/auth forms (forms fall back to line prompts on stdin off-TTY) (default true)
  -q, --jq string              Filter and transform output using a jq expression (e.g., '.name', '.items[] | .id')
      --no-interactive         Disable all interactive features (auto-prompting, explorer auto-launch, TUI forms)
      --no-keyring             Never read or write the OS keychain; store secrets in the config file instead (env: SHADEFORM_NO_KEYRING)
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

* `shade instances update-model --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade instances update-model --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `shade instances update-model --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
