//go:build !windows

package winutil

func ListRunningApps() ([]RunningApp, error) {
	return nil, nil
}
