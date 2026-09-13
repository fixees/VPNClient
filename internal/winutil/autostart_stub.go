//go:build !windows

package winutil

func EnableAutostart() error  { return ErrNotWindows }
func DisableAutostart() error { return nil }
func IsAutostartEnabled() bool { return false }
