//go:build !windows

package winutil

func IsAdmin() bool { return true }

func RelaunchAsAdmin(args ...string) error {
	return ErrNotWindows
}

func EnsureAdmin() (bool, error) { return true, nil }

func RequireAdminForTUN(tunEnabled bool) error { return nil }

func MessageBox(title, text string, isError bool) {}

func ActivateMainWindow(windowTitle string) bool { return false }

func HandleSecondInstance(windowTitle string) {}
