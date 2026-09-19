//go:build !windows

package winutil

// KillProcessesWithImagePath is a no-op outside Windows.
func KillProcessesWithImagePath(exePath string) (int, error) {
	_ = exePath
	return 0, nil
}
