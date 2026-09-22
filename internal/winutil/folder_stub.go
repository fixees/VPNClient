//go:build !windows

package winutil

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenFolder reveals dir in the platform file manager.
func OpenFolder(dir string) error {
	dir = filepath.Clean(dir)
	if dir == "" {
		return fmt.Errorf("empty folder path")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}
