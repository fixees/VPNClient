package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ExtractFS writes embedded core assets into destDir (typically AppData/.../core).
// Files are replaced when missing or when the content hash differs.
func ExtractFS(fsys fs.FS, root string, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(filepath.FromSlash(path))
		if strings.EqualFold(base, "readme.md") || strings.EqualFold(base, ".gitkeep") {
			return nil
		}
		raw, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		dest := filepath.Join(destDir, base)
		if sameFile(dest, raw) {
			return nil
		}
		tmp := dest + ".tmp"
		if err := os.WriteFile(tmp, raw, 0o755); err != nil {
			return err
		}
		if err := os.Rename(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		return nil
	})
}

func sameFile(path string, want []byte) bool {
	st, err := os.Stat(path)
	if err != nil || st.Size() != int64(len(want)) {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	sum := h.Sum(nil)
	wantSum := sha256.Sum256(want)
	return hex.EncodeToString(sum) == hex.EncodeToString(wantSum[:])
}
