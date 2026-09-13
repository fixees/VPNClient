//go:build !windows

package winutil

import "fmt"

func IsAdmin() bool { return false }

func RelaunchAsAdmin(args ...string) error {
	return ErrNotWindows
}

func RequireAdminForTUN(tunEnabled bool) error {
	if tunEnabled {
		return fmt.Errorf("%w: TUN unsupported on this OS build", ErrNotWindows)
	}
	return nil
}
