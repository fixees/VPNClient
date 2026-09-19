package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"myinternetvpn/client/internal/defaults"
)

// Checker looks for a newer Windows package on a GitHub Release tag (e.g. latest-main).
type Checker struct {
	Owner      string
	Repo       string
	Tag        string
	HTTPClient *http.Client
	UserAgent  string
}

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Release struct {
	TagName string         `json:"tag_name"`
	Name    string         `json:"name"`
	Assets  []ReleaseAsset `json:"assets"`
}

type AvailableUpdate struct {
	Tag      string
	Asset    ReleaseAsset
	Download string
}

func NewGitHub(owner, repo, tag string) *Checker {
	if tag == "" {
		tag = defaults.UpdateTag
	}
	return &Checker{
		Owner: owner,
		Repo:  repo,
		Tag:   tag,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: defaults.UpdaterUA,
	}
}

func (c *Checker) FetchRelease() (Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", c.Owner, c.Repo, c.Tag)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Release{}, err
	}
	if resp.StatusCode >= 300 {
		return Release{}, fmt.Errorf("github release %s: status %d: %s", c.Tag, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return Release{}, err
	}
	return rel, nil
}

// FindWindowsZip picks the first MyInternetVPN windows amd64 zip asset.
func (c *Checker) FindWindowsZip(rel Release) (ReleaseAsset, error) {
	for _, a := range rel.Assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, "windows") && strings.HasSuffix(name, ".zip") {
			return a, nil
		}
		if strings.HasPrefix(name, strings.ToLower(defaults.ReleasePrefix)+"-") && strings.HasSuffix(name, ".zip") {
			return a, nil
		}
	}
	return ReleaseAsset{}, fmt.Errorf("no windows zip asset in release %s", rel.TagName)
}

// Check returns an update when remote asset name/tag differs from currentVersion marker.
func (c *Checker) Check(currentVersion string) (AvailableUpdate, bool, error) {
	rel, err := c.FetchRelease()
	if err != nil {
		return AvailableUpdate{}, false, err
	}
	asset, err := c.FindWindowsZip(rel)
	if err != nil {
		return AvailableUpdate{}, false, err
	}
	if VersionMatches(currentVersion, asset.Name, rel.TagName) {
		return AvailableUpdate{}, false, nil
	}
	return AvailableUpdate{Tag: rel.TagName, Asset: asset, Download: asset.BrowserDownloadURL}, true, nil
}

// VersionMatches reports whether the running build already corresponds to the
// remote release asset (full/short git sha in filename, or identical tag).
func VersionMatches(currentVersion, assetName, tag string) bool {
	current := strings.TrimSpace(currentVersion)
	if current == "" {
		return false
	}
	// Local/dev builds are never considered "same as release".
	lower := strings.ToLower(current)
	if lower == "dev" || lower == "development" || lower == "local" {
		return false
	}
	assetLower := strings.ToLower(assetName)
	curLower := strings.ToLower(current)
	if strings.EqualFold(strings.TrimSpace(tag), current) {
		return true
	}
	if strings.Contains(assetLower, curLower) {
		return true
	}
	// Git short SHAs embedded via -ldflags / CI (7–12 hex chars).
	if isGitSHA(current) {
		short := curLower
		if len(short) > 12 {
			short = short[:12]
		}
		if len(short) >= 7 && strings.Contains(assetLower, short[:7]) {
			return true
		}
		if strings.Contains(assetLower, short) {
			return true
		}
	}
	return false
}

func isGitSHA(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

// DownloadTo stores the asset into destDir and returns the local zip path.
func (c *Checker) DownloadTo(asset ReleaseAsset, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	path := filepath.Join(destDir, asset.Name)
	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}
