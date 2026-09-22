//go:build windows

package winutil

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// OpenFolder opens dir in Explorer.
func OpenFolder(dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" {
		return fmt.Errorf("empty folder path")
	}
	cmd := exec.Command("explorer.exe", dir)
	HideConsole(cmd)
	return cmd.Start()
}
