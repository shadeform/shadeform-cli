## shade templates update

Update a template

### Synopsis

Update an existing template

```
shade templates update [template-id] [flags]
```

### Examples

```
  shade templates update --template-id <id> --name <value> --public=true
```

### Options

```
      --alert string                  Send an email alert when a date or a spend threshold is reached.
      --author string                 Who published the template. Up to 50 characters.
      --auto-delete string            Delete the instance automatically when a date or a spend threshold is reached.
      --body string                   Request body as JSON (alternative to individual flags). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
      --description string            What the template launches and when to use it. Up to 10000 characters.
  -e, --envs string                   Environment variables to set on the instance for every shell session. Up to 20. (JSON array)
  -f, --featured string               True for templates Shadeform curates and lists under GET /v1/templates/featured. Read-only.
  -h, --help                          help for update
      --id string                     The ID of the template, assigned by Shadeform. Ignored on save.
      --is-account-owner string       True when the template belongs to the requesting account. Read-only.
      --launch-configuration string   What to run automatically once the instance is active: a docker container or a startup script. See https://docs.shadeform.ai/getting-started/concepts#launch-configuration.
      --logo string                   URL of a logo shown with the template. Up to 200 characters.
  -m, --metadata string               Free-form labels attached to the template, such as the model or framework it runs. (JSON array)
      --name string                   The name of the template. Alphanumeric characters, spaces, underscores and dashes, up to 100 characters. [required]
      --networking string             Network configuration, currently UFW firewall rules applied at boot.
  -p, --public                        True makes the template visible to every Shadeform account; false keeps it private to yours. [required]
      --schema                        Print the exact JSON Schema of the request body and exit
      --tags string                   Custom, searchable tags. Up to 200 characters each. (JSON array)
  -t, --template-id string            The ID of the template, from the id field of GET /v1/templates. (or pass it as the [template-id] argument)
  -v, --volume-mount string           How attached storage volumes are mounted onto the instance's file system.
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

* [shade templates](shade_templates.md)	 - Save and reuse launch configurations as templates

### Machine interface

* `shade templates update --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade templates update --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `shade templates update --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
