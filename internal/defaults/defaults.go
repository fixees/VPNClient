// Package defaults holds shared product identity and tunable defaults.
// Prefer these over scattered magic strings/numbers.
package defaults

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Product identity (single source of truth for UI, paths, mutex, firewall, updates).
const (
	ProductName   = "MyInternetVPN"
	ProductExe    = "MyInternetVPN.exe"
	DataDirName   = "MyInternetVPN"
	WindowTitle   = "MyInternetVPN"
	UserAgent     = "MyInternetVPN/2.0"
	UpdaterUA     = "MyInternetVPN-Updater/2.0"
	MutexName     = "Global\\MyInternetVPN_SingleInstance"
	AutostartName = "MyInternetVPN"
)

// Firewall rule name prefixes (Windows).
const (
	KillSwitchBlockRule = "MyInternetVPN_KillSwitch_BlockAll"
	KillSwitchAllowRule = "MyInternetVPN_KillSwitch_AllowVPN"
	DNSLeakRule         = "MyInternetVPN_DNSLeak_Block53"
)

// Network / controller defaults.
const (
	MixedPort       = 7890
	ControllerAddr  = "127.0.0.1:9090"
	ProxyGroup      = "PROXY"
	AutoGroup       = "AUTO"
	DefaultMode     = "rule"
	DefaultLogLevel = "info"
	DefaultVPNIface = "Meta"
	UpdateTag       = "latest-main"
	UpdateOwner     = "fixees"
	UpdateRepo      = "VPNClient"
)

// Timing.
const (
	APITimeout            = 3 * time.Second
	TrafficSampleTimeout  = 800 * time.Millisecond
	ReadyTimeout          = 8 * time.Second
	StopGrace             = 2 * time.Second
	HealthPollInterval    = 150 * time.Millisecond
	DefaultHealthInterval = 30 * time.Second
	StatusCacheTTL        = 2 * time.Second
	VersionCacheTTL       = 30 * time.Second
	URLTestTimeoutMS      = 5000
	URLTestIntervalSec    = 300
	DefaultSubIntervalMin = 360
	ReconnectBackoff      = 5 * time.Second
)

// DNS / latency probe defaults used in generated Clash config.
var (
	DNSNameservers = []string{"8.8.8.8", "1.1.1.1"}
	DNSFallbacks   = []string{"tls://1.1.1.1:853"}
	URLTestURL     = "https://www.gstatic.com/generate_204"
)

// Config generation defaults (mihomo).
const (
	DNSFakeIPRange     = "198.18.0.1/16"
	DNSEnhancedMode    = "fake-ip"
	DefaultTUNStack    = "system"
	DefaultBypassGEOIP = "CN"
	WARPEndpoint       = "engage.cloudflareclient.com:2408"
	WARPPublicKey      = "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="
	WARPLocalAddress   = "172.16.0.2/32"
	WARPProxyName      = "WARP"
)

// NewLocalSecret returns a random secret for the local controller.
func NewLocalSecret() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte("fallback-local-secret"))[:32]
	}
	return hex.EncodeToString(b[:])
}
