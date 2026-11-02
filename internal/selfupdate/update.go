package selfupdate

import (
	"context"
	"fmt"
	"runtime"
)

// Updater resolves a release for this platform and turns it into a verified
// binary. It does not touch the file system; Replace does that.
type Updater struct {
	Client *Client
	GOOS   string
	GOARCH string
}

// NewUpdater returns an updater for the running platform.
func NewUpdater(c *Client) *Updater {
	return &Updater{Client: c, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
}

// Plan is a resolved release plus the two assets the update needs.
type Plan struct {
	Release      *Release
	AssetName    string
	AssetURL     string
	ChecksumsURL string
}

// Resolve looks up the release for tag ("" means latest) and checks it ships
// an archive and a checksum manifest for this platform.
func (u *Updater) Resolve(ctx context.Context, tag string) (*Plan, error) {
	asset, err := AssetName(u.GOOS, u.GOARCH)
	if err != nil {
		return nil, err
	}
	var rel *Release
	if tag == "" {
		rel, err = u.Client.Latest(ctx)
	} else {
		rel, err = u.Client.Release(ctx, tag)
	}
	if err != nil {
		return nil, err
	}
	assetURL, ok := rel.Assets[asset]
	if !ok {
		return nil, fmt.Errorf("release %s has no %s", rel.Tag, asset)
	}
	sumsURL, ok := rel.Assets[ChecksumsFile]
	if !ok {
		return nil, fmt.Errorf("release %s has no %s; refusing to install an unverified binary", rel.Tag, ChecksumsFile)
	}
	return &Plan{Release: rel, AssetName: asset, AssetURL: assetURL, ChecksumsURL: sumsURL}, nil
}

// Fetch downloads the archive and the checksum manifest, verifies the
// archive and returns the binary inside it. A mismatch returns an error
// wrapping ErrChecksumMismatch and nothing is written anywhere.
func (u *Updater) Fetch(ctx context.Context, p *Plan) ([]byte, error) {
	sums, err := u.Client.Download(ctx, p.ChecksumsURL)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", ChecksumsFile, err)
	}
	want, ok := ParseChecksums(sums)[p.AssetName]
	if !ok {
		return nil, fmt.Errorf("%s in release %s does not list %s", ChecksumsFile, p.Release.Tag, p.AssetName)
	}
	archive, err := u.Client.Download(ctx, p.AssetURL)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", p.AssetName, err)
	}
	if err := VerifySHA256(archive, want); err != nil {
		return nil, fmt.Errorf("%s: %w", p.AssetName, err)
	}
	return ExtractBinary(p.AssetName, archive, BinaryName(u.GOOS))
}
