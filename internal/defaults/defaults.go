// Package defaults holds shared product identity and tunable defaults.
// Prefer these over scattered magic strings/numbers.
package defaults

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

// Product identity (single source of truth for UI, paths, mutex, firewall, updates).
const (
	ProductName       = "Мой VPN"
	ProductClientName = "Мой VPN Client"
	ProductExe        = "MyInternetVPN.exe" // release / update artifact name (ASCII)
	ReleasePrefix     = "MyInternetVPN"     // GitHub zip asset prefix
	DataDirName       = "MyInternetVPN"     // keep stable AppData path
	WindowTitle       = "Мой VPN"
	UserAgent         = "MyVPN/2.0"
	UpdaterUA         = "MyVPN-Updater/2.0"
	MutexName         = "Global\\MyInternetVPN_SingleInstance"
	AutostartName     = "Мой VPN"
	// Custom URL schemes (Windows protocol handlers), Incy-style deep links.
	URLScheme    = "myvpn"
	URLSchemeAlt = "myinternetvpn"
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
	// Mihomo PUT /configs?force=true rebuilds TUN/DNS/rules; 3s is too short on large profiles.
	ReloadConfigTimeout   = 45 * time.Second
	// Mihomo pushes /traffic once per tick (default 1s); allow >1s for one-shot reads.
	TrafficSampleTimeout  = 2 * time.Second
	TrafficStreamInterval = 500 // ms; passed as ?interval=
	TrafficFreshness      = 3 * time.Second
	ReadyTimeout          = 8 * time.Second
	StopGrace             = 2 * time.Second
	HealthPollInterval    = 150 * time.Millisecond
	DefaultHealthInterval = 30 * time.Second
	StatusCacheTTL        = 2 * time.Second
	VersionCacheTTL       = 30 * time.Second
	ModeCacheTTL          = 5 * time.Second
	PublicIPCacheTTL      = 45 * time.Second
	URLTestTimeoutMS      = 5000
	URLTestIntervalSec    = 300
	DelayAPITimeoutMaxMS  = 30000 // mihomo parses timeout as int16
	DefaultSubIntervalMin = 360
	ReconnectBackoff      = 5 * time.Second
	// Must stay ≤ DelayAPITimeoutMaxMS (int16); larger values → HTTP 400 "Body invalid".
	AutoGroupDelayFloorMS = 20000
)

// DNS / latency probe defaults used in generated Clash config.
var (
	DNSNameservers = []string{"8.8.8.8", "1.1.1.1"}
	DNSFallbacks   = []string{"tls://1.1.1.1:853"}
	URLTestURL     = "https://www.gstatic.com/generate_204"
)

// Ping / URL-test method IDs.
const (
	PingProxyHTTPGet  = "proxy-http-get"  // via mihomo /proxies/{name}/delay
	PingProxyHTTPHead = "proxy-http-head" // HEAD through local mixed proxy
	PingTCP           = "tcp"             // TCP dial to node server:port
	PingHTTPGet       = "http-get"        // direct HTTP GET to URL (no proxy)
	PingICMP          = "icmp"            // ICMP echo to node server
	DefaultPingMethod = PingProxyHTTPGet
)

// URLTestPresets maps UI preset keys to probe URLs.
var URLTestPresets = map[string]string{
	"gstatic":    "https://www.gstatic.com/generate_204",
	"cloudflare": "https://www.cloudflare.com/cdn-cgi/trace",
	"apple":      "http://captive.apple.com/hotspot-detect.html",
}

// ResolveURLTestURL returns preset URL or custom trimmed value.
func ResolveURLTestURL(preset, custom string) string {
	if u, ok := URLTestPresets[strings.ToLower(strings.TrimSpace(preset))]; ok {
		return u
	}
	custom = strings.TrimSpace(custom)
	if custom != "" {
		return custom
	}
	return URLTestURL
}

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
	// WARPModeViaProxy: exit via WARP, WireGuard dials through selected PROXY node (Hiddify default).
	WARPModeViaProxy = "via-proxy"
	// WARPModeProxyViaWARP: selected node dials through WARP first.
	WARPModeProxyViaWARP = "proxy-via-warp"
)

// NewLocalSecret returns a random secret for the local controller.
func NewLocalSecret() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte("fallback-local-secret"))[:32]
	}
	return hex.EncodeToString(b[:])
}
