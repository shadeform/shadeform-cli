package cli

import (
	"runtime/debug"
	"testing"
)

func TestModuleVersion(t *testing.T) {
	const generated = "0.3.1"

	withVersion := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: "github.com/shadeform/shadeform-cli", Version: v}}, true
		}
	}

	tests := []struct {
		name string
		read func() (*debug.BuildInfo, bool)
		want string
	}{
		{
			name: "go install of a release newer than the generated number",
			read: withVersion("v0.3.2"),
			want: "0.3.2",
		},
		{
			name: "go install of a later minor",
			read: withVersion("v0.10.0"),
			want: "0.10.0",
		},
		{
			name: "go install of a prerelease",
			read: withVersion("v1.0.0-rc.1"),
			want: "1.0.0-rc.1",
		},
		{
			name: "go install of the generated version",
			read: withVersion("v0.3.1"),
			want: generated,
		},
		{
			name: "module version older than the generated number",
			read: withVersion("v0.3.0"),
			want: generated,
		},
		{
			name: "go build without version control",
			read: withVersion("(devel)"),
			want: generated,
		},
		{
			name: "no module version",
			read: withVersion(""),
			want: generated,
		},
		{
			name: "go build of an untagged commit after a release",
			read: withVersion("v0.3.2-0.20261124183000-ff9fbd98a1b2"),
			want: generated,
		},
		{
			name: "go build of an untagged commit with no release before it",
			read: withVersion("v0.0.0-20261124183000-ff9fbd98a1b2"),
			want: generated,
		},
		{
			name: "go build of an untagged commit after a prerelease",
			read: withVersion("v1.0.0-rc.1.0.20261124183000-ff9fbd98a1b2"),
			want: generated,
		},
		{
			name: "go build of a tagged commit with local changes",
			read: withVersion("v0.3.2+dirty"),
			want: generated,
		},
		{
			name: "go build of an untagged commit with local changes",
			read: withVersion("v0.3.2-0.20261124183000-ff9fbd98a1b2+dirty"),
			want: generated,
		},
		{
			name: "module version that is not a version",
			read: withVersion("main"),
			want: generated,
		},
		{
			name: "no build info",
			read: func() (*debug.BuildInfo, bool) { return nil, false },
			want: generated,
		},
		{
			name: "nil build info",
			read: func() (*debug.BuildInfo, bool) { return nil, true },
			want: generated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := moduleVersion(generated, tt.read); got != tt.want {
				t.Errorf("moduleVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
