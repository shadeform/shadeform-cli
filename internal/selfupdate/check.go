package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// CacheFileName sits beside config.yaml and remembers the last check, so the
// passive notice costs one anonymous GitHub request a day at most.
const CacheFileName = "update-check.yaml"

// DefaultInterval is how long a cached answer is trusted.
const DefaultInterval = 24 * time.Hour

// NoCheckEnv disables the passive check when set to anything non-empty.
const NoCheckEnv = "SHADEFORM_NO_UPDATE_CHECK"

// Cache is the on-disk record. Nothing in it identifies the user.
type Cache struct {
	CheckedAt time.Time `yaml:"checked_at"`
	Latest    string    `yaml:"latest,omitempty"`
}

// CachePath returns the cache file's location given the config file's path.
func CachePath(configPath string) string {
	if configPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(configPath), CacheFileName)
}

// LoadCache reads the cache; a missing or unreadable file is an empty cache.
func LoadCache(path string) Cache {
	var c Cache
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Cache{}
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Cache{}
	}
	return c
}

// SaveCache writes the cache, creating the directory if needed.
func SaveCache(path string, c Cache) error {
	if path == "" {
		return errors.New("no cache path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Gate describes the situation after a command ran; ShouldNotify decides
// from it alone, so the rule is testable without a terminal.
type Gate struct {
	// CurrentVersion is the running version; a notice compares against it.
	CurrentVersion string
	// ReleaseBuild is false for go build / go install binaries, which have
	// no release to be behind.
	ReleaseBuild bool
	// CommandFailed suppresses the notice so it never trails an error.
	CommandFailed bool
	// InteractiveTTY: stdout and stderr are both terminals; the notice is for
	// people, not for pipes.
	InteractiveTTY bool
	// AgentMode: coding agents get structured output only.
	AgentMode bool
	// OutputFormat must be "pretty"; -o json and friends stay clean.
	OutputFormat string
	// JQ is set when --jq was given, which also means machine output.
	JQ bool
	// DryRun: --dry-run promises no network.
	DryRun bool
	// SkipCommand marks commands where a notice is noise: update itself,
	// shell completion scripts.
	SkipCommand bool
	// Getenv reads CI and SHADEFORM_NO_UPDATE_CHECK; nil means os.Getenv.
	Getenv func(string) string
}

// ShouldNotify applies every condition from the ticket. All must hold.
func ShouldNotify(g Gate) bool {
	getenv := g.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	switch {
	case !g.ReleaseBuild,
		g.CommandFailed,
		!g.InteractiveTTY,
		g.AgentMode,
		g.OutputFormat != "pretty",
		g.JQ,
		g.DryRun,
		g.SkipCommand,
		getenv(NoCheckEnv) != "",
		getenv("CI") != "":
		return false
	}
	_, err := ParseVersion(g.CurrentVersion)
	return err == nil
}

// Checker answers "what is the latest release" from the cache when it is
// fresh and from GitHub otherwise.
type Checker struct {
	Client    *Client
	CachePath string
	Interval  time.Duration
	Now       func() time.Time
}

// Latest returns the latest release tag, or "" when it is not known. A
// failed request is recorded so it is not retried until the interval
// passes; being offline must not slow every command down.
func (c *Checker) Latest(ctx context.Context) string {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	interval := c.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	cache := LoadCache(c.CachePath)
	if !cache.CheckedAt.IsZero() && now().Sub(cache.CheckedAt) < interval {
		return cache.Latest
	}
	cache.CheckedAt = now().UTC()
	if rel, err := c.Client.Latest(ctx); err == nil && rel != nil {
		cache.Latest = rel.Tag
	}
	_ = SaveCache(c.CachePath, cache)
	return cache.Latest
}

// Notice is the single line printed when a newer release exists.
func Notice(current, latest string) string {
	cur, err := ParseVersion(current)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("A new version of shade is available (%s -> %s). Run: shade update", cur, latest)
}

// Notify prints the notice to w if the gate allows it and a newer release
// is known. It never returns an error: this is a courtesy, not a step.
func (c *Checker) Notify(ctx context.Context, g Gate, w io.Writer) {
	if !ShouldNotify(g) {
		return
	}
	latest := c.Latest(ctx)
	if latest == "" || !IsNewer(latest, g.CurrentVersion) {
		return
	}
	if line := Notice(g.CurrentVersion, latest); line != "" {
		fmt.Fprintln(w, line)
	}
}
