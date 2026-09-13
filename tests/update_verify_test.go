package tests

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"myinternetvpn/client/internal/update"
)

func TestVerifyPackageSHA256AndEd25519(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "app.zip")
	content := []byte("package-bytes")
	if err := os.WriteFile(zipPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	sum, err := update.FileSHA256(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	shaPath := zipPath + ".sha256"
	if err := os.WriteFile(shaPath, []byte(sum+"  app.zip\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := update.SignFile(zipPath, priv)
	if err != nil {
		t.Fatal(err)
	}
	sigPath := zipPath + ".sig"
	if err := os.WriteFile(sigPath, []byte(sig), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := update.VerifyPackage(zipPath, shaPath, sigPath, pub); err != nil {
		t.Fatalf("VerifyPackage: %v", err)
	}

	// tamper
	_ = os.WriteFile(zipPath, []byte("tampered"), 0o644)
	if err := update.VerifyPackage(zipPath, shaPath, sigPath, pub); err == nil {
		t.Fatal("expected verify failure after tamper")
	}
}

func TestFindCompanionAsset(t *testing.T) {
	rel := update.Release{
		Assets: []update.ReleaseAsset{
			{Name: "MyInternetVPN-windows-amd64-abc.zip"},
			{Name: "MyInternetVPN-windows-amd64-abc.zip.sha256"},
			{Name: "MyInternetVPN-windows-amd64-abc.zip.sig"},
		},
	}
	sha, err := update.FindCompanionAsset(rel, "MyInternetVPN-windows-amd64-abc.zip", ".sha256")
	if err != nil || sha.Name == "" {
		t.Fatalf("sha: %v %#v", err, sha)
	}
	sig, err := update.FindCompanionAsset(rel, "MyInternetVPN-windows-amd64-abc.zip", ".sig")
	if err != nil || sig.Name == "" {
		t.Fatalf("sig: %v %#v", err, sig)
	}
}
