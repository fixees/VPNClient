//go:build windows

package winutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const autostartKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const autostartValue = "MyInternetVPN"

// EnableAutostart registers the current executable to run at user logon.
func EnableAutostart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	cmd := fmt.Sprintf("\"%s\" --autostart", exe)
	return key.SetStringValue(autostartValue, cmd)
}

// DisableAutostart removes the Run registry value.
func DisableAutostart() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(autostartValue)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

// IsAutostartEnabled reports whether the Run value exists.
func IsAutostartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	val, _, err := key.GetStringValue(autostartValue)
	return err == nil && strings.TrimSpace(val) != ""
}
