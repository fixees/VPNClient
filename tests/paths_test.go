package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"myinternetvpn/client/internal/paths"
)

func TestPathsNewWithRoots(t *testing.T) {
	tmp := t.TempDir()
	r := paths.NewWithRoots(filepath.Join(tmp, "data"), filepath.Join(tmp, "resources"))

	if err := r.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if _, err := os.Stat(r.DataDir()); err != nil {
		t.Fatalf("data dir missing: %v", err)
	}
	if r.ProfilesFile() != filepath.Join(tmp, "data", "profiles.json") {
		t.Fatalf("unexpected profiles path: %s", r.ProfilesFile())
	}

	wantName := "mihomo"
	if runtime.GOOS == "windows" {
		wantName = "mihomo.exe"
	}
	want := filepath.Join(tmp, "data", "core", wantName)
	if r.CoreBinary() != want {
		t.Fatalf("core binary = %s, want %s", r.CoreBinary(), want)
	}
}
