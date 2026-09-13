package geo

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Default MetaCubeX asset URLs (mihomo-compatible).
const (
	DefaultGeoIPURL   = "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb"
	DefaultGeoSiteURL = "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat"
)

// Ensure makes sure GeoIP/GeoSite databases exist in workDir.
// Prefer already-extracted bundled files; download only if still missing.
func Ensure(workDir string) error {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	pairs := []struct {
		name string
		url  string
	}{
		{"geoip.metadb", DefaultGeoIPURL},
		{"geosite.dat", DefaultGeoSiteURL},
	}
	for _, p := range pairs {
		dst := filepath.Join(workDir, p.name)
		if st, err := os.Stat(dst); err == nil && st.Size() > 0 {
			continue
		}
		if err := download(p.url, dst); err != nil {
			return fmt.Errorf("geo asset %s: %w", p.name, err)
		}
	}
	return nil
}

func download(url, dst string) error {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}
