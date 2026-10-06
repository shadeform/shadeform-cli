## shade instances list-types

List available instance types

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
  -a, --available                    Filter the instance type results by availability. Applies to individual availability entries, so an instance type whose region is available only as spot is still returned with just its spot entry; filter by 'rental_type' client-side if you only want on-demand.
  -c, --cloud string                 Filter the instance type results by cloud.
  -g, --gpu-type string              Filter the instance type results by gpu type.
  -h, --help                         help for list-types
  -n, --num-gpus string              Filter the instance type results by the number of gpus.
  -r, --region string                Filter the instance type results by region.
      --shade-instance-type string   Filter the instance type results by the shade instance type.
      --sort string                  Sort the order of the instance type results. (options: price)
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

* `shade instances list-types --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade instances list-types --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
