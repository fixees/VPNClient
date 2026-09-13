package main

import (
	"fmt"
	"os"

	"myinternetvpn/client/internal/update"
)

// Usage: go run ./scripts/sign-release.go <zip> <private-key-file> <out.sig>
func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: sign-release.go <zip> <private-key-file> <out.sig>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}
	priv, err := update.ParsePrivateKey(string(raw))
	if err != nil {
		panic(err)
	}
	sig, err := update.SignFile(os.Args[1], priv)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[3], []byte(sig+"\n"), 0o644); err != nil {
		panic(err)
	}
	fmt.Println(os.Args[3])
}
