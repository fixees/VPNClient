//go:build !windows

package winutil

func AcquireSingleInstance() (bool, error) { return true, nil }

func ReleaseSingleInstance() {}
