//go:build windows

package winutil

import (
	"os/exec"
	"strings"
	"sync"
)

var (
	driverCacheMu sync.Mutex
	driverCache   = map[string]bool{}
)

// DriverServiceRunning reports whether a Windows driver/service appears running.
// Used as a soft signal for WinDivert-based tools (never sufficient alone).
func DriverServiceRunning(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	driverCacheMu.Lock()
	if v, ok := driverCache[name]; ok {
		driverCacheMu.Unlock()
		return v
	}
	driverCacheMu.Unlock()

	cmd := exec.Command("sc.exe", "query", name)
	HideConsole(cmd)
	out, err := cmd.CombinedOutput()
	running := false
	if err == nil {
		text := strings.ToLower(string(out))
		running = strings.Contains(text, "running")
	}
	driverCacheMu.Lock()
	driverCache[name] = running
	driverCacheMu.Unlock()
	return running
}
