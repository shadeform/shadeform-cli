package selfupdate

import (
	"path/filepath"
	"runtime/debug"
	"strings"
)

// Source is how the running binary got onto this machine. Only release
// archives (the install script, or a manual download) are updated in place;
// the others have their own upgrade path and would be clobbered by ours.
type Source int

const (
	// SourceRelease is a binary from a GitHub release archive.
	SourceRelease Source = iota
	// SourceHomebrew lives under a Homebrew prefix.
	SourceHomebrew
	// SourceGoInstall was built by `go install module@version`.
	SourceGoInstall
	// SourceDev was built from a checkout without release ldflags.
	SourceDev
)

func (s Source) String() string {
	switch s {
	case SourceRelease:
		return "release"
	case SourceHomebrew:
		return "homebrew"
	case SourceGoInstall:
		return "go-install"
	case SourceDev:
		return "dev"
	}
	return "unknown"
}

// Updatable reports whether `shade update` may replace this binary.
func (s Source) Updatable() bool { return s == SourceRelease }

// Advice is what to run instead when the binary is not updatable.
func (s Source) Advice() string {
	switch s {
	case SourceHomebrew:
		return "this shade was installed with Homebrew; run: brew upgrade shade"
	case SourceGoInstall:
		return "this shade was installed with go install; run: go install github.com/" + Repo + "/cmd/shade@latest"
	case SourceDev:
		return "this shade was built from source; rebuild it from the repository instead"
	}
	return ""
}

// DetectSource classifies the binary at exe. releaseBuild is whether the
// release ldflags were present at build time (GoReleaser sets them; plain
// go build and go install do not).
func DetectSource(exe string, releaseBuild bool) Source {
	return detectSource(exe, releaseBuild, debug.ReadBuildInfo)
}

func detectSource(exe string, releaseBuild bool, readBuildInfo func() (*debug.BuildInfo, bool)) Source {
	p := filepath.ToSlash(exe)
	if strings.Contains(p, "/Cellar/") || strings.Contains(p, "/homebrew/") || strings.Contains(p, "/linuxbrew/") {
		return SourceHomebrew
	}
	if releaseBuild {
		return SourceRelease
	}
	if info, ok := readBuildInfo(); ok {
		v := info.Main.Version
		if v != "" && v != "(devel)" {
			return SourceGoInstall
		}
	}
	return SourceDev
}
