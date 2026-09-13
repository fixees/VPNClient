package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// FileSHA256 returns the hex-encoded SHA-256 of path.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ReadExpectedHash loads a hash from a .sha256 file.
// Accepts either bare hex or `hex  filename` (sha256sum format).
func ReadExpectedHash(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(raw))
	if line == "" {
		return "", fmt.Errorf("empty hash file")
	}
	fields := strings.Fields(line)
	return strings.ToLower(fields[0]), nil
}

// VerifyFile compares file digest to expected hex (case-insensitive).
func VerifyFile(path, expectedHex string) error {
	if expectedHex == "" {
		return fmt.Errorf("expected hash is empty")
	}
	got, err := FileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, strings.TrimSpace(expectedHex)) {
		return fmt.Errorf("integrity check failed for %s: got %s want %s", path, got, expectedHex)
	}
	return nil
}

// VerifyBeside checks path against path+".sha256" when the sidecar exists.
// If sidecar is missing, returns (false, nil) meaning "skipped".
func VerifyBeside(path string) (checked bool, err error) {
	sidecar := path + ".sha256"
	if _, err := os.Stat(sidecar); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	expected, err := ReadExpectedHash(sidecar)
	if err != nil {
		return true, err
	}
	return true, VerifyFile(path, expected)
}
