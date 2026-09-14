//go:build !windows

package winutil

func RegisterURLProtocols() error { return nil }

func PendingDeepLinkFile() string { return "" }

func WritePendingDeepLink(string) error { return nil }

func TakePendingDeepLink() (string, bool) { return "", false }

func HandleSecondInstance(windowTitle, deepLink string) {}
