## shade instances create

Create an instance

### Synopsis

Create a new GPU instance. Our create API is designed to be asynchronous, so the response will be a CreateResponse object with a status of "creating". We then have a process that will pick it up and create it. You can poll the /instances/{id}/info endpoint to check the status of the instance.

```
shade instances create [flags]
```

### Examples

```
  shade instances create --cloud <value> --name <value> --region <value> --shade-instance-type <value>
```

### Options

```
      --alert string                  Send an email alert when a date or a spend threshold is reached.
      --auto-delete string            Delete the instance automatically when a date or a spend threshold is reached.
      --body string                   Request body as JSON (alternative to individual flags). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
  -c, --cloud string                  The underlying cloud provider, as listed by GET /v1/instances/types. See https://docs.shadeform.ai/getting-started/concepts#cloud-cloud-provider. [required]
      --enable-metrics string         Enable GPU and system metrics collection on the instance where the instance type supports it.
      --envs string                   Environment variables to set on the instance for every shell session. Up to 20. (JSON array)
  -h, --help                          help for create
  -l, --launch-configuration string   What to run automatically once the instance is active: a docker container or a startup script. See https://docs.shadeform.ai/getting-started/concepts#launch-configuration.
      --name string                   The name of the instance. Alphanumeric characters, underscores and dashes, up to 50 characters. [required]
      --networking string             Network configuration, currently UFW firewall rules applied at boot.
      --os string                     The operating system image, from the os_options of the instance type. Defaults to the instance type's default image.
      --region string                 The region of the cloud provider, as listed in the availability entries of GET /v1/instances/types. [required]
      --rental-type string            How the instance is rented: on_demand (default) or spot. Spot instances run at the discounted spot hourly price but are interruptible: Shadeform may reclaim the capacity at any time, at which point the instance is terminated with status_details set to preempted. See https://docs.shadeform.ai/guides/spotinstances.
      --schema                        Print the exact JSON Schema of the request body and exit
      --shade-cloud string            True launches the instance in Shade Cloud, billed through Shadeform; false launches it in your linked cloud account. Required for AWS and Azure, where only linked accounts are supported. See https://docs.shadeform.ai/getting-started/concepts#shade-cloud.
      --shade-instance-type string    The Shadeform standardized instance type, as listed by GET /v1/instances/types. See https://docs.shadeform.ai/getting-started/concepts#shade-instance-type-and-cloud-instance-type. [required]
      --ssh-key-id string             The ID of the SSH key to install, from GET /v1/sshkeys. Defaults to the account's default key.
      --tags string                   Custom, searchable tags. Up to 200 characters each. (JSON array)
      --template-id string            The ID of a template whose launch configuration, auto-delete, alert, volume mount, tags, environment variables and networking are applied to the instance. Fields set directly on this request take precedence.
      --volume-ids string             IDs of storage volumes to attach at launch. Currently at most one volume. (JSON array)
      --volume-mount string           How attached storage volumes are mounted onto the instance's file system.
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

* `shade instances create --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `shade instances create --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `shade instances create --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
