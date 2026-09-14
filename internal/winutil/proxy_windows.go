//go:build windows

package winutil

import (
	"fmt"
	"strconv"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const internetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// SystemProxySnapshot stores previous proxy settings for restore.
type SystemProxySnapshot struct {
	ProxyEnable   uint32
	ProxyServer   string
	ProxyOverride string
}

// EnableSystemProxy points WinINET proxy at host:port.
func EnableSystemProxy(host string, port int) (SystemProxySnapshot, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return SystemProxySnapshot{}, err
	}
	defer key.Close()

	snap, err := readProxySnapshot(key)
	if err != nil {
		return SystemProxySnapshot{}, err
	}

	server := host + ":" + strconv.Itoa(port)
	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return snap, err
	}
	if err := key.SetStringValue("ProxyServer", server); err != nil {
		return snap, err
	}
	if err := key.SetStringValue("ProxyOverride", "<local>"); err != nil {
		return snap, err
	}
	notifyProxyChanged()
	return snap, nil
}

func readProxySnapshot(key registry.Key) (SystemProxySnapshot, error) {
	snap := SystemProxySnapshot{}
	if v, _, err := key.GetIntegerValue("ProxyEnable"); err == nil {
		snap.ProxyEnable = uint32(v)
	}
	if v, _, err := key.GetStringValue("ProxyServer"); err == nil {
		snap.ProxyServer = v
	}
	if v, _, err := key.GetStringValue("ProxyOverride"); err == nil {
		snap.ProxyOverride = v
	}
	return snap, nil
}

// ReadSystemProxy returns the current WinINET proxy settings.
func ReadSystemProxy() (SystemProxySnapshot, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.QUERY_VALUE)
	if err != nil {
		return SystemProxySnapshot{}, err
	}
	defer key.Close()
	return readProxySnapshot(key)
}

// RestoreSystemProxy writes a previous snapshot back.
func RestoreSystemProxy(snap SystemProxySnapshot) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	_ = key.SetDWordValue("ProxyEnable", snap.ProxyEnable)
	_ = key.SetStringValue("ProxyServer", snap.ProxyServer)
	_ = key.SetStringValue("ProxyOverride", snap.ProxyOverride)
	notifyProxyChanged()
	return nil
}

// DisableSystemProxy turns ProxyEnable off.
func DisableSystemProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return err
	}
	notifyProxyChanged()
	return nil
}

// ClearOurSystemProxy disables WinINET proxy when it still points at our mixed port
// (orphan after crash / hard power-off). Returns true when settings were changed.
func ClearOurSystemProxy(host string, port int) (bool, error) {
	cur, err := ReadSystemProxy()
	if err != nil {
		return false, err
	}
	if cur.ProxyEnable == 0 || !IsOurProxyServer(cur.ProxyServer, host, port) {
		return false, nil
	}
	if err := DisableSystemProxy(); err != nil {
		return false, err
	}
	return true, nil
}

func notifyProxyChanged() {
	wininet := windows.NewLazySystemDLL("wininet.dll")
	proc := wininet.NewProc("InternetSetOptionW")
	const (
		internetOptionSettingsChanged = 39
		internetOptionRefresh         = 37
	)
	_, _, _ = proc.Call(0, uintptr(internetOptionSettingsChanged), 0, 0)
	_, _, _ = proc.Call(0, uintptr(internetOptionRefresh), 0, 0)
}

func FormatProxyError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("system proxy: %w", err)
}
