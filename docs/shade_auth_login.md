## shade auth login

Sign in through your browser and store an API key for this machine

### Synopsis

Sign in to Shadeform from the terminal.

shade opens your browser to confirm a short code. Once you approve, it
creates an API key for this machine - named "shade CLI (<hostname>)" so
you can recognise it under Settings > API Keys - and stores it in the OS
keychain (or ~/.config/shade/config.yaml with --no-keyring). Nothing else
is stored: the browser sign-in is used once and forgotten.

Signing in again on the same machine replaces the previous key. To use a
key you already have instead, pass --api-key; nothing is opened and no key
is created.

On a machine without a browser, open the printed URL anywhere and enter the
code there.

```
shade auth login [flags]
```

### Examples

```
  shade auth login
  shade auth login --no-browser
  shade auth login --name "shade CLI (build box)"
  shade auth login --api-key sk_live_xxxxxxxx
```

### Options

```
  -h, --help                 help for login
      --name string          Name for the API key as shown in the dashboard (default "shade CLI (<hostname>)")
      --no-browser           Print the sign-in URL and code instead of opening a browser
      --permissions string   Comma-separated permissions for the key: platform_read, platform_write, billing_read, billing_write (default "platform_read,platform_write")
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

* [shade auth](shade_auth.md)	 - Manage authentication credentials

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
