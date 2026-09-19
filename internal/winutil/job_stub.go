//go:build !windows

package winutil

// AssignProcessToChildKillJob is a no-op outside Windows.
func AssignProcessToChildKillJob(pid int) error {
	_ = pid
	return nil
}
