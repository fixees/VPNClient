package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// Generates an Ed25519 keypair for release signing.
// Usage: go run ./scripts/gen-update-keys.go
func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	root, _ := os.Getwd()
	pubPath := filepath.Join(root, "resources", "update", "ed25519_public.key")
	privPath := filepath.Join(root, "secrets", "update_ed25519_private.key")
	_ = os.MkdirAll(filepath.Dir(pubPath), 0o755)
	_ = os.MkdirAll(filepath.Dir(privPath), 0o755)
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(pubPath, []byte(pubB64+"\n"), 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(privPath, []byte(privB64+"\n"), 0o600); err != nil {
		panic(err)
	}
	fmt.Println("public: ", pubPath)
	fmt.Println("private:", privPath)
	fmt.Println("Keep the private key offline / in CI secrets. Never commit secrets/.")
}
