package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Repo is the GitHub repository releases are published from.
const Repo = "shadeform/shadeform-cli"

const (
	defaultAPIBase = "https://api.github.com"
	// maxDownload bounds every response body read here. Release archives
	// are a few MB; anything near this is wrong.
	maxDownload = 200 << 20
)

// ErrNoRelease is returned when GitHub reports no release for the request
// (no releases at all, or an unknown tag).
var ErrNoRelease = errors.New("no such release")

// Release is the part of a GitHub release the updater needs.
type Release struct {
	Tag        string
	Prerelease bool
	// Assets maps an asset file name to its browser download URL.
	Assets map[string]string
}

// Client talks to the GitHub REST API anonymously. The only thing it sends
// is a User-Agent naming the CLI version, which GitHub requires.
type Client struct {
	HTTP      *http.Client
	APIBase   string
	UserAgent string
}

// NewClient returns a client with a 3 second timeout, the budget the passive
// check is allowed to spend. `shade update` reuses it with a longer one.
func NewClient(version string) *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 3 * time.Second},
		APIBase:   defaultAPIBase,
		UserAgent: "shade/" + strings.TrimPrefix(version, "v"),
	}
}

// Latest returns the release GitHub marks as latest (never a prerelease or
// a draft).
func (c *Client) Latest(ctx context.Context) (*Release, error) {
	return c.get(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", c.apiBase(), Repo))
}

// Release returns the release for a tag, with or without the leading v.
func (c *Client) Release(ctx context.Context, tag string) (*Release, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, errors.New("empty release tag")
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return c.get(ctx, fmt.Sprintf("%s/repos/%s/releases/tags/%s", c.apiBase(), Repo, tag))
}

func (c *Client) get(ctx context.Context, url string) (*Release, error) {
	body, status, err := c.fetch(ctx, url, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, ErrNoRelease
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned HTTP %d for %s", status, url)
	}
	var raw struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing GitHub release: %w", err)
	}
	if raw.TagName == "" || raw.Draft {
		return nil, ErrNoRelease
	}
	rel := &Release{Tag: raw.TagName, Prerelease: raw.Prerelease, Assets: make(map[string]string, len(raw.Assets))}
	for _, a := range raw.Assets {
		rel.Assets[a.Name] = a.URL
	}
	return rel, nil
}

// Download fetches a release asset by URL.
func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	body, status, err := c.fetch(ctx, url, "application/octet-stream")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("download of %s returned HTTP %d", url, status)
	}
	return body, nil
}

func (c *Client) fetch(ctx context.Context, url, accept string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", accept)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(body) > maxDownload {
		return nil, resp.StatusCode, fmt.Errorf("response from %s exceeds %d bytes", url, maxDownload)
	}
	return body, resp.StatusCode, nil
}

func (c *Client) apiBase() string {
	if c.APIBase != "" {
		return strings.TrimRight(c.APIBase, "/")
	}
	return defaultAPIBase
}
