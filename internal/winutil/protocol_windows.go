//go:build windows

package winutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"myinternetvpn/client/internal/defaults"

	"golang.org/x/sys/windows/registry"
)

// RegisterURLProtocols registers myvpn:// and myinternetvpn:// for the current user.
func RegisterURLProtocols() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	var errs []string
	for _, scheme := range []string{defaults.URLScheme, defaults.URLSchemeAlt} {
		if err := registerOneProtocol(scheme, exe); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", scheme, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

func registerOneProtocol(scheme, exe string) error {
	keyPath := `Software\Classes\` + scheme
	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetStringValue("", "URL:"+defaults.ProductName+" Protocol"); err != nil {
		return err
	}
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cmdKey.Close()
	cmd := fmt.Sprintf("\"%s\" \"%%1\"", exe)
	return cmdKey.SetStringValue("", cmd)
}

// PendingDeepLinkFile is the handoff path for a second-instance deep link.
func PendingDeepLinkFile() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, defaults.DataDirName, "pending-deeplink.txt")
}

// WritePendingDeepLink stores a URL for the running instance to consume.
func WritePendingDeepLink(link string) error {
	link = strings.TrimSpace(link)
	if link == "" {
		return nil
	}
	path := PendingDeepLinkFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(link), 0o600)
}

// TakePendingDeepLink reads and deletes a pending deep link, if any.
func TakePendingDeepLink() (string, bool) {
	path := PendingDeepLinkFile()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	_ = os.Remove(path)
	link := strings.TrimSpace(string(b))
	if link == "" {
		return "", false
	}
	return link, true
}

// HandleSecondInstance activates the running UI. When deepLink is set, it is
// handed off via pending file (no MessageBox).
func HandleSecondInstance(windowTitle, deepLink string) {
	deepLink = strings.TrimSpace(deepLink)
	if deepLink != "" {
		_ = WritePendingDeepLink(deepLink)
	}
	_ = ActivateExistingClient(windowTitle)
}
