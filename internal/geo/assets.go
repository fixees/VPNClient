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

// Ensure copies or downloads GeoIP/GeoSite databases into workDir.
// Prefers local files from resourceDir when present.
func Ensure(workDir, resourceDir string) error {
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
		src := filepath.Join(resourceDir, "core", p.name)
		if st, err := os.Stat(src); err == nil && st.Size() > 0 {
			if err := copyFile(src, dst); err != nil {
				return err
			}
			continue
		}
		if err := download(p.url, dst); err != nil {
			return fmt.Errorf("geo asset %s: %w", p.name, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
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
