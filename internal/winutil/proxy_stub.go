//go:build !windows

package winutil

type SystemProxySnapshot struct {
	ProxyEnable   uint32
	ProxyServer   string
	ProxyOverride string
}

func EnableSystemProxy(host string, port int) (SystemProxySnapshot, error) {
	return SystemProxySnapshot{}, ErrNotWindows
}

func ReadSystemProxy() (SystemProxySnapshot, error) {
	return SystemProxySnapshot{}, ErrNotWindows
}

func RestoreSystemProxy(snap SystemProxySnapshot) error { return ErrNotWindows }

func DisableSystemProxy() error { return ErrNotWindows }

func ClearOurSystemProxy(host string, port int) (bool, error) { return false, nil }

func FormatProxyError(err error) error {
	if err == nil {
		return nil
	}
	return err
}
