package winutil

import (
	"fmt"
	"os"
	"strings"
)

// HasCLIFlag reports whether argv contains an exact flag (e.g. "--autostart").
func HasCLIFlag(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, a := range os.Args[1:] {
		if a == name {
			return true
		}
	}
	return false
}

// IsOurProxyServer reports whether a WinINET ProxyServer value points at host:port.
// Accepts plain "127.0.0.1:7890" and "http=127.0.0.1:7890;https=…".
func IsOurProxyServer(server, host string, port int) bool {
	if strings.TrimSpace(server) == "" || strings.TrimSpace(host) == "" || port <= 0 {
		return false
	}
	needle := strings.ToLower(fmt.Sprintf("%s:%d", host, port))
	s := strings.ToLower(strings.TrimSpace(server))
	if s == needle {
		return true
	}
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if i := strings.IndexByte(part, '='); i >= 0 {
			part = strings.TrimSpace(part[i+1:])
		}
		if part == needle {
			return true
		}
	}
	return false
}
