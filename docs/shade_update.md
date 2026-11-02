## shade update

Update shade to the latest release

### Synopsis

Update shade to the latest release from GitHub.

shade downloads the archive for this platform from
https://github.com/shadeform/shadeform-cli/releases, verifies it against the
release's checksums.txt, and replaces the running binary in place. The old
binary is only swapped out once the new one is fully written and verified.

A binary installed with Homebrew or go install is not touched; the command
prints the matching upgrade command instead.

shade also mentions a newer release, once a day at most, after a command
finishes at an interactive terminal. Set SHADEFORM_NO_UPDATE_CHECK=1 to turn that
off. The check is a single anonymous request to api.github.com and sends
nothing about you or your account.

```
shade update [flags]
```

### Examples

```
  shade update
  shade update --check
  shade update --version v0.3.1
```

### Options

```
      --check            Only report whether a newer release exists; do not install
  -h, --help             help for update
      --version string   Install this release (e.g. v0.3.1) instead of the latest
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

* [shade](shade.md)	 - Shadeform API: Shadeform is a single API and platform for deploying and managing cloud GPUs

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
