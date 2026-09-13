package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FindCompanionAsset finds an asset whose name equals baseName+suffix or contains it.
func FindCompanionAsset(rel Release, zipName, suffix string) (ReleaseAsset, error) {
	want := zipName + suffix
	for _, a := range rel.Assets {
		if strings.EqualFold(a.Name, want) {
			return a, nil
		}
	}
	// fallback: any asset ending with suffix that shares zip stem
	stem := strings.TrimSuffix(zipName, filepath.Ext(zipName))
	for _, a := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), strings.ToLower(suffix)) &&
			strings.Contains(strings.ToLower(a.Name), strings.ToLower(stem)) {
			return a, nil
		}
	}
	return ReleaseAsset{}, fmt.Errorf("companion asset %s not found", want)
}

// DownloadVerified downloads zip + .sha256 + .sig and verifies before returning zip path.
func (c *Checker) DownloadVerified(rel Release, zipAsset ReleaseAsset, destDir, publicKeyB64 string) (string, error) {
	shaAsset, err := FindCompanionAsset(rel, zipAsset.Name, ".sha256")
	if err != nil {
		return "", fmt.Errorf("release is missing checksum sidecar: %w", err)
	}
	sigAsset, err := FindCompanionAsset(rel, zipAsset.Name, ".sig")
	if err != nil {
		return "", fmt.Errorf("release is missing signature sidecar: %w", err)
	}
	pub, err := ParsePublicKey(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("public key: %w", err)
	}

	zipPath, err := c.DownloadTo(zipAsset, destDir)
	if err != nil {
		return "", err
	}
	shaPath, err := c.DownloadTo(shaAsset, destDir)
	if err != nil {
		return "", err
	}
	sigPath, err := c.DownloadTo(sigAsset, destDir)
	if err != nil {
		return "", err
	}
	if err := VerifyPackage(zipPath, shaPath, sigPath, pub); err != nil {
		_ = os.Remove(zipPath)
		return "", err
	}
	return zipPath, nil
}

// DownloadBytes is a small helper used by tests.
func DownloadBytes(client *http.Client, url, userAgent string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
