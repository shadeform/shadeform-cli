package selfupdate

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"v0.3.1", "v0.3.1", true},
		{"0.3.1", "v0.3.1", true},
		{" v1.2.3 ", "v1.2.3", true},
		{"v1.0.0-rc.1", "v1.0.0-rc.1", true},
		{"v1.0.0+build.5", "v1.0.0", true},
		{"v1.0.0-beta+exp", "v1.0.0-beta", true},
		{"", "", false},
		{"v1.2", "", false},
		{"v1.2.3.4", "", false},
		{"v01.2.3", "", false},
		{"v1.2.x", "", false},
		{"v1.2.3-", "", false},
		{"latest", "", false},
	}
	for _, c := range cases {
		v, err := ParseVersion(c.in)
		if (err == nil) != c.ok {
			t.Errorf("ParseVersion(%q): err=%v, want ok=%v", c.in, err, c.ok)
			continue
		}
		if c.ok && v.String() != c.want {
			t.Errorf("ParseVersion(%q) = %s, want %s", c.in, v, c.want)
		}
	}
}

func TestCompare(t *testing.T) {
	order := []string{
		"v0.9.9",
		"v1.0.0-alpha",
		"v1.0.0-alpha.1",
		"v1.0.0-alpha.beta",
		"v1.0.0-beta",
		"v1.0.0-beta.2",
		"v1.0.0-beta.11",
		"v1.0.0-rc.1",
		"v1.0.0",
		"v1.0.1",
		"v1.1.0",
		"v2.0.0",
	}
	for i := range order {
		for j := range order {
			a, _ := ParseVersion(order[i])
			b, _ := ParseVersion(order[j])
			got := Compare(a, b)
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got != want {
				t.Errorf("Compare(%s, %s) = %d, want %d", order[i], order[j], got, want)
			}
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.3.1", "0.3.0", true},
		{"v0.3.1", "v0.3.1", false},
		{"v0.3.0", "0.3.1", false},
		{"v0.4.0", "0.3.9", true},
		{"v1.0.0", "1.0.0-rc.1", true},
		{"v1.0.0-rc.1", "1.0.0", false},
		{"garbage", "0.3.1", false},
		{"v0.3.1", "(devel)", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"darwin", "arm64", "shade_Darwin_arm64.tar.gz"},
		{"darwin", "amd64", "shade_Darwin_x86_64.tar.gz"},
		{"linux", "amd64", "shade_Linux_x86_64.tar.gz"},
		{"linux", "arm64", "shade_Linux_arm64.tar.gz"},
		{"windows", "amd64", "shade_Windows_x86_64.zip"},
		{"windows", "arm64", "shade_Windows_arm64.zip"},
	}
	for _, c := range cases {
		got, err := AssetName(c.goos, c.goarch)
		if err != nil || got != c.want {
			t.Errorf("AssetName(%s, %s) = %q, %v; want %q", c.goos, c.goarch, got, err, c.want)
		}
	}
	if _, err := AssetName("freebsd", "amd64"); err == nil {
		t.Error("AssetName(freebsd) should fail")
	}
	if _, err := AssetName("linux", "386"); err == nil {
		t.Error("AssetName(linux/386) should fail")
	}
}

func TestParseChecksums(t *testing.T) {
	data := []byte("abc123  shade_Darwin_arm64.tar.gz\nDEF456 *shade_Windows_x86_64.zip\n\nnot-a-line\n")
	got := ParseChecksums(data)
	if got["shade_Darwin_arm64.tar.gz"] != "abc123" {
		t.Errorf("tar.gz digest = %q", got["shade_Darwin_arm64.tar.gz"])
	}
	if got["shade_Windows_x86_64.zip"] != "def456" {
		t.Errorf("zip digest = %q (binary-mode marker and case should be normalised)", got["shade_Windows_x86_64.zip"])
	}
	if len(got) != 2 {
		t.Errorf("parsed %d entries, want 2", len(got))
	}
}
