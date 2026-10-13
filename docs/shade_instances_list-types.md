## shade instances list-types

List instance types

### Synopsis

Return all the GPU instance types with their corresponding availability and specs.

Every `availability` entry carries a `rental_type` (`on_demand` or `spot`). Instance types that offer spot list each region twice, once per rental type, and the two entries can differ in `available`. If your integration keys on `region` or checks whether any region is available, filter to `rental_type: on_demand` first to keep the previous behavior. See the [Spot Instances guide](/guides/spotinstances) for details.

```
shade instances list-types [flags]
```

### Examples

```
  shade instances list-types
```

### Options

```
  -a, --available string             "true" keeps only available entries, "false" only unavailable ones. Any other value is ignored.
  -c, --cloud string                 Only return instance types on this cloud (case-insensitive).
  -g, --gpu-type string              Only return instance types with this GPU type (case-insensitive).
  -h, --help                         help for list-types
  -n, --num-gpus int                 A non-zero value returns only instance types with exactly that GPU count. 0 disables the filter, and so does omitting the parameter - the two are indistinguishable, so this filter cannot select instance types with no GPUs.
  -r, --region string                Only return availability entries in this Shade region (case-insensitive). Instance types left with no matching region are dropped from the response entirely.
      --shade-cloud string           "true" excludes the hyperscalers (AWS, Azure). Any other value is ignored.
      --shade-instance-type string   Only return this Shade instance type (case-insensitive).
      --sort string                  Ordering, not filtering: "price" sorts ascending by on-demand hourly price, removing nothing. Any other value is ignored.
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

* `shade instances list-types --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade instances list-types --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
