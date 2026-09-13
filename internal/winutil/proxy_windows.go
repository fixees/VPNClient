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
