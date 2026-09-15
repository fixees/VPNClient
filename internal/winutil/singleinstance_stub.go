//go:build !windows

package winutil

func AcquireSingleInstance() (bool, error) { return true, nil }

func ReleaseSingleInstance() {}

func InstanceAlreadyRunning() bool { return false }

func ActivateExistingClient(string) bool { return false }

func ActivateByExeNames([]string) bool { return false }
