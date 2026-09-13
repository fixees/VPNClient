package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// ReadSHA256File parses a sha256sum-style file (hex or "hex  filename").
func ReadSHA256File(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(raw))
	if line == "" {
		return "", fmt.Errorf("empty checksum file")
	}
	fields := strings.Fields(line)
	return strings.ToLower(fields[0]), nil
}

// FileSHA256 returns hex digest of path.
func FileSHA256(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// VerifySHA256 compares file digest to expected hex.
func VerifySHA256(path, expectedHex string) error {
	got, err := FileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, strings.TrimSpace(expectedHex)) {
		return fmt.Errorf("checksum mismatch for %s: got %s want %s", path, got, expectedHex)
	}
	return nil
}

// VerifySHA256File loads expected digest from sidecar and verifies path.
func VerifySHA256File(path, checksumPath string) error {
	expected, err := ReadSHA256File(checksumPath)
	if err != nil {
		return err
	}
	return VerifySHA256(path, expected)
}

// ParsePublicKey decodes a base64 Ed25519 public key.
func ParsePublicKey(b64 string) (ed25519.PublicKey, error) {
	b64 = strings.TrimSpace(b64)
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key size %d", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// ParsePrivateKey decodes a base64 Ed25519 private key.
func ParsePrivateKey(b64 string) (ed25519.PrivateKey, error) {
	b64 = strings.TrimSpace(b64)
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid ed25519 private key size %d", len(raw))
	}
	return ed25519.PrivateKey(raw), nil
}

// SignFile creates a base64 Ed25519 signature over the raw file bytes.
func SignFile(path string, priv ed25519.PrivateKey) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, raw)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifySignatureFile verifies path against a base64 signature file using pub.
func VerifySignatureFile(path, sigPath string, pub ed25519.PublicKey) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sigB64, err := os.ReadFile(sigPath)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(pub, raw, sig) {
		return fmt.Errorf("ed25519 signature verification failed for %s", path)
	}
	return nil
}

// VerifyPackage requires SHA-256 sidecar and Ed25519 signature.
func VerifyPackage(zipPath, checksumPath, sigPath string, pub ed25519.PublicKey) error {
	if err := VerifySHA256File(zipPath, checksumPath); err != nil {
		return fmt.Errorf("checksum: %w", err)
	}
	if pub == nil {
		return fmt.Errorf("update public key is not configured")
	}
	if err := VerifySignatureFile(zipPath, sigPath, pub); err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	return nil
}
