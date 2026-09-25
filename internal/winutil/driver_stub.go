//go:build !windows

package winutil

// DriverServiceRunning is always false outside Windows.
func DriverServiceRunning(name string) bool {
	_ = name
	return false
}
