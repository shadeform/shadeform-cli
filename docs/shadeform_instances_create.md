## shadeform instances create

Create an instance

### Synopsis

Create a new GPU instance. Our create API is designed to be asynchronous, so the response will be a CreateResponse object with a status of "creating". We then have a process that will pick it up and create it. You can poll the /instances/{id}/info endpoint to check the status of the instance.

```
shadeform instances create [flags]
```

### Examples

```
  shadeform instances create --cloud hyperstack --region canada-1 --shade-instance-type A6000 --shade-cloud=true --name cool-gpu-server
```

### Options

```
      --alert string                  Set a date or spend threshold to receive an email alert
      --auto-delete string            Set a date or spend threshold to automatically delete the instance
      --body string                   Request body as JSON (alternative to individual flags). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
  -c, --cloud string                  Specifies the underlying cloud provider. See this [explanation](/getting-started/concepts#cloud-cloud-provider) for more details. [required]
  -e, --envs string                   List of environment variable name and values to automatically add to the instance (JSON array)
  -h, --help                          help for create
  -l, --launch-configuration string   Defines automatic actions after the instance becomes active.
  -n, --name string                   The name of the instance [required]
      --os string                     The operating system of the instance.
      --region string                 Specifies the region. [required]
      --rental-type string            How the instance is rented. Defaults to on_demand. Spot instances run at a discounted hourly price but are interruptible: Shadeform may reclaim the capacity at any time, at which point the instance is terminated. Spot requires an instance type that supports it (see the availability entries tagged rental_type spot in the instance types response) and shade_cloud set to true. (options: on_demand, spot) (default "on_demand")
      --schema                        Print the exact JSON Schema of the request body and exit
      --shade-cloud                   Specifies if the instance is launched in [Shade Cloud](/getting-started/concepts#shade-cloud) or in a linked cloud account. [required]
      --shade-instance-type string    The Shadeform standardized instance type. See this [explanation](/getting-started/concepts#shade-instance-type-and-cloud-instance-type) for more details. [required]
      --ssh-key-id string             The ID of the SSH Key.
      --tags stringArray              Add custom, searchable tags to instances.
      --template-id string            The ID of the template to use for this instance
      --volume-ids stringArray        List of volume IDs to be mounted. Currently only supports 1 volume at a time.
      --volume-mount string           Settings for mounting volumes onto file systems
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

* [shadeform instances](shadeform_instances.md)	 - Create, inspect, update, restart and delete GPU instances

### Machine interface

* `shadeform instances create --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shadeform instances create --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `shadeform instances create --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
