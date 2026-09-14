//go:build !windows

package winutil

import "os/exec"

// HideConsole is a no-op on non-Windows.
func HideConsole(cmd *exec.Cmd) {}
