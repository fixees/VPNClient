package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/profiles"

	"gopkg.in/yaml.v3"
)

// BuildInput drives runtime mihomo config generation.
type BuildInput struct {
	Profile          profiles.Profile
	MixedPort        int
	ControllerURL    string
	Secret           string
	Mode             string
	TUN              bool
	TUNStack         string
	VPNInterface     string // TUN device-name + kill-switch interface
	AllowLAN         bool
	IPv6             bool
	LogLevel         string
	ProxyGroup       string
	URLTestURL       string
	URLTestSec       int
	BypassLAN        bool
	BypassGEOIP      string
	RouteDirect      string // multiline: domains / IPs / regexp → DIRECT
	RouteBlock       string // multiline → REJECT
	RouteProxy       string // multiline → PROXY group
	RouteWhitelist   bool   // MATCH → REJECT; only listed DIRECT/PROXY allowed
	AppRouteMode     string // off | whitelist | blacklist
	AppRouteList     string // multiline PROCESS-NAME / paths
	// AppWhitelistKeepDirect keeps MATCH,DIRECT when whitelist mode had apps
	// but compat stripped them all (avoid collapsing to full-tunnel MATCH,PROXY).
	AppWhitelistKeepDirect bool
	DNSEnhancedMode        string
	DNSNameservers   []string
	DNSFallbacks     []string
	DNSFakeIPRange   string
	Sniffer          bool
	TCPConcurrent    bool
	UnifiedDelay     bool
	WARPEnabled      bool
	WARPMode         string // via-proxy | proxy-via-warp
	WARPPrivateKey   string
	WARPLocalAddress string
	WARPEndpoint     string
	WARPPublicKey    string
	WARPCleanIP      string
	WARPPort         int
	WARPNoiseCount   string
	WARPNoiseMode    string
	WARPNoiseSize    string
	WARPNoiseDelay   string
}

// Build produces a Clash Meta / mihomo YAML document.
func Build(in BuildInput) ([]byte, error) {
	if in.Profile.Name == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	if len(in.Profile.Proxies) == 0 {
		return nil, fmt.Errorf("profile has no proxies")
	}
	if in.MixedPort <= 0 {
		in.MixedPort = defaults.MixedPort
	}
	if in.ControllerURL == "" {
		in.ControllerURL = defaults.ControllerAddr
	}
	if in.Mode == "" {
		in.Mode = defaults.DefaultMode
	}
	if in.LogLevel == "" {
		in.LogLevel = defaults.DefaultLogLevel
	}
	if in.ProxyGroup == "" {
		in.ProxyGroup = defaults.ProxyGroup
	}
	if in.URLTestURL == "" {
		in.URLTestURL = defaults.URLTestURL
	}
	if in.URLTestSec <= 0 {
		in.URLTestSec = defaults.URLTestIntervalSec
	}
	if in.TUNStack == "" {
		in.TUNStack = defaults.DefaultTUNStack
	}
	if in.DNSEnhancedMode == "" {
		in.DNSEnhancedMode = defaults.DNSEnhancedMode
	}
	if len(in.DNSNameservers) == 0 {
		in.DNSNameservers = append([]string(nil), defaults.DNSNameservers...)
	}
	if len(in.DNSFallbacks) == 0 {
		in.DNSFallbacks = append([]string(nil), defaults.DNSFallbacks...)
	}
	if in.DNSFakeIPRange == "" {
		in.DNSFakeIPRange = defaults.DNSFakeIPRange
	}

	appRouting := AppRoutingEnabled(in)
	appWhitelist := AppWhitelistActive(in)
	// fake-ip breaks DIRECT apps that still traverse TUN (Discord RTC/QUIC, etc.).
	// App split-tunnel always has some PROCESS → DIRECT path, so prefer redir-host.
	dnsMode := in.DNSEnhancedMode
	if appRouting {
		dnsMode = "redir-host"
	}

	proxyNames := make([]string, 0, len(in.Profile.Proxies)+1)
	proxies := make([]map[string]any, 0, len(in.Profile.Proxies)+1)
	for i, p := range in.Profile.Proxies {
		name, _ := p["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("proxy at index %d missing name", i)
		}
		if name == defaults.WARPProxyName {
			continue // reserved for protection layer
		}
		node := map[string]any(p)
		if in.WARPEnabled && warpMode(in) == defaults.WARPModeProxyViaWARP {
			node = cloneProxyMap(p)
			node["dialer-proxy"] = defaults.WARPProxyName
		}
		proxyNames = append(proxyNames, name)
		proxies = append(proxies, node)
	}

	if in.WARPEnabled {
		warp, err := buildWARPProxy(in)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, warp)
	}

	selectProxies := append([]string{defaults.AutoGroup}, proxyNames...)
	rules := buildRules(in)

	dns := map[string]any{
		"enable":        true,
		"enhanced-mode": dnsMode,
		"fake-ip-range": in.DNSFakeIPRange,
		"nameserver":    in.DNSNameservers,
		"fallback":      in.DNSFallbacks,
		"ipv6":          in.IPv6,
	}
	if appRouting {
		// Resolve DNS using the same PROCESS/MATCH rules as traffic.
		// mihomo requires proxy-server-nameserver when respect-rules is on
		// (used to resolve proxy node hostnames without rule recursion).
		dns["respect-rules"] = true
		dns["proxy-server-nameserver"] = append([]string(nil), in.DNSNameservers...)
		// DIRECT apps (browser etc.) should use the OS resolver — DoT fallback via
		// TUN is a common cause of “HTML loads, CSS/CDN never arrives”.
		dns["direct-nameserver"] = []string{"system"}
	}
	if appWhitelist {
		// Most traffic is DIRECT: skip slow TLS DNS fallbacks that queue behind TUN.
		dns["fallback"] = []string{}
	}

	doc := map[string]any{
		"mixed-port":          in.MixedPort,
		"allow-lan":           in.AllowLAN,
		"mode":                in.Mode,
		"log-level":           in.LogLevel,
		"ipv6":                in.IPv6,
		"external-controller": in.ControllerURL,
		"secret":              in.Secret,
		"tcp-concurrent":      in.TCPConcurrent,
		"unified-delay":       in.UnifiedDelay,
		"dns":                 dns,
		"proxies":             proxies,
		"proxy-groups": []map[string]any{
			{
				"name":    in.ProxyGroup,
				"type":    "select",
				"proxies": selectProxies,
			},
			{
				"name":      defaults.AutoGroup,
				"type":      "url-test",
				"proxies":   proxyNames,
				"url":       in.URLTestURL,
				"interval":  in.URLTestSec,
				"lazy":      false,
				"tolerance": 50,
			},
		},
		"rules": rules,
	}

	if appRouting {
		// Needed so PROCESS-NAME / PROCESS-PATH rules resolve under TUN.
		doc["find-process-mode"] = "always"
	}

	if in.TUN {
		device := strings.TrimSpace(in.VPNInterface)
		if device == "" {
			device = defaults.DefaultVPNIface
		}
		tun := map[string]any{
			"enable":                true,
			"device-name":           device,
			"stack":                 in.TUNStack,
			"auto-route":            true,
			"auto-detect-interface": true,
			"dns-hijack":            []string{"any:53"},
		}
		if appWhitelist {
			// Helps UDP/TCP NAT reuse for many parallel DIRECT browser connections.
			tun["endpoint-independent-nat"] = true
		}
		doc["tun"] = tun
	}

	// Sniffer on every DIRECT browser flow under TUN is expensive and often stalls CDNs.
	// With app split-tunnel alone, keep it off. But custom site lists need SNI sniffing:
	// Electron/Cursor use DoH, so DOMAIN rules never see a DNS mapping without the sniffer.
	// override-destination=false: rewriting the dial IP from SNI breaks Cloudflare Access
	// and similar TLS flows (ERR_SSL_PROTOCOL_ERROR) while DNS mapping is enough for rules.
	if in.Sniffer && (!appRouting || HasCustomSiteRules(in)) {
		doc["sniffer"] = map[string]any{
			"enable":               true,
			"parse-pure-ip":        true,
			"force-dns-mapping":    true,
			"override-destination": false,
			"skip-domain": []string{
				"+.cloudflareaccess.com",
				"cloudflareaccess.com",
			},
			"sniff": map[string]any{
				"TLS": map[string]any{
					"ports": []any{443, "8443"},
				},
				"HTTP": map[string]any{
					"ports": []any{80, "8080-8880"},
				},
			},
		}
	}

	// Prefer local geo databases when present in workdir (-d).
	doc["geodata-mode"] = true

	return yaml.Marshal(doc)
}

func buildRules(in BuildInput) []string {
	rules := make([]string, 0, 12)
	if in.BypassLAN {
		for _, cidr := range []string{
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"169.254.0.0/16",
			"224.0.0.0/4",
		} {
			rules = append(rules, "IP-CIDR,"+cidr+",DIRECT,no-resolve")
		}
		if in.IPv6 {
			for _, cidr := range []string{"FC00::/7", "FE80::/10", "::1/128"} {
				rules = append(rules, "IP-CIDR6,"+cidr+",DIRECT,no-resolve")
			}
		}
	}
	// Custom lists: block → direct → force-proxy, then per-app process rules,
	// then GEOIP (unless domain whitelist), then MATCH.
	rules = AppendCustomRules(rules, in)
	rules = AppendAppProcessRules(rules, in)
	appMode := NormalizeAppRouteMode(in.AppRouteMode)
	if !in.RouteWhitelist && appMode != AppRouteWhitelist {
		if geo := strings.ToUpper(strings.TrimSpace(in.BypassGEOIP)); geo != "" && geo != "OFF" && geo != "NONE" {
			rules = append(rules, "GEOIP,"+geo+",DIRECT")
		}
	}
	// App whitelist: only listed apps use VPN; everything else goes DIRECT.
	if AppWhitelistActive(in) {
		rules = append(rules, "MATCH,DIRECT")
		return rules
	}
	// Domain whitelist: everything not explicitly allowed is blocked.
	if in.RouteWhitelist {
		rules = append(rules, "MATCH,REJECT")
		return rules
	}
	// WARP as exit layer (via-proxy): MATCH → WARP, WireGuard dials through PROXY.
	// proxy-via-warp: MATCH → PROXY (nodes dial through WARP).
	if in.WARPEnabled && warpMode(in) == defaults.WARPModeViaProxy {
		rules = append(rules, "MATCH,"+defaults.WARPProxyName)
	} else {
		rules = append(rules, "MATCH,"+in.ProxyGroup)
	}
	return rules
}

func warpMode(in BuildInput) string {
	m := strings.ToLower(strings.TrimSpace(in.WARPMode))
	if m == defaults.WARPModeProxyViaWARP {
		return defaults.WARPModeProxyViaWARP
	}
	return defaults.WARPModeViaProxy
}

// HasCustomSiteRules reports whether user site lists will emit DOMAIN/IP rules.
func HasCustomSiteRules(in BuildInput) bool {
	return strings.TrimSpace(in.RouteBlock) != "" ||
		strings.TrimSpace(in.RouteDirect) != "" ||
		strings.TrimSpace(in.RouteProxy) != ""
}

func cloneProxyMap(p profiles.ProxyNode) map[string]any {
	out := make(map[string]any, len(p)+1)
	for k, v := range p {
		out[k] = v
	}
	return out
}

func buildWARPProxy(in BuildInput) (map[string]any, error) {
	key := strings.TrimSpace(in.WARPPrivateKey)
	if key == "" {
		return nil, fmt.Errorf("WARP включён, но не задан private key — сгенерируйте конфиг в настройках")
	}
	endpoint := strings.TrimSpace(in.WARPEndpoint)
	if endpoint == "" {
		endpoint = defaults.WARPEndpoint
	}
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		host = endpoint
		portStr = "2408"
	}
	if clean := strings.TrimSpace(in.WARPCleanIP); clean != "" && !strings.EqualFold(clean, "auto") {
		host = clean
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		port = 2408
	}
	if in.WARPPort > 0 {
		port = in.WARPPort
	}
	pub := strings.TrimSpace(in.WARPPublicKey)
	if pub == "" {
		pub = defaults.WARPPublicKey
	}
	local := strings.TrimSpace(in.WARPLocalAddress)
	if local == "" {
		local = defaults.WARPLocalAddress
	}
	ip := strings.TrimSuffix(local, "/32")
	ip = strings.TrimSuffix(ip, "/128")

	doc := map[string]any{
		"name":        defaults.WARPProxyName,
		"type":        "wireguard",
		"server":      host,
		"port":        port,
		"ip":          ip,
		"private-key": key,
		"public-key":  pub,
		"udp":         true,
		"mtu":         1280,
	}
	// Hiddify default: establish WARP tunnel through the selected VPN node.
	if warpMode(in) == defaults.WARPModeViaProxy {
		doc["dialer-proxy"] = in.ProxyGroup
	}
	if opt := amneziaOption(in); opt != nil {
		doc["amnezia-wg-option"] = opt
	}
	return doc, nil
}

func amneziaOption(in BuildInput) map[string]any {
	// Optional DPI noise (AmneziaWG). Applied when count/size look configured.
	jc := parseRangeMid(in.WARPNoiseCount, 0)
	jmin, jmax := parseRangePair(in.WARPNoiseSize, 0, 0)
	if jc <= 0 && jmin <= 0 && jmax <= 0 {
		return nil
	}
	if jc <= 0 {
		jc = 3
	}
	if jmin <= 0 {
		jmin = 10
	}
	if jmax < jmin {
		jmax = jmin + 20
	}
	return map[string]any{
		"jc":   jc,
		"jmin": jmin,
		"jmax": jmax,
		"s1":   0,
		"s2":   0,
		"h1":   1,
		"h2":   2,
		"h3":   3,
		"h4":   4,
	}
}

func parseRangeMid(s string, fallback int) int {
	a, b := parseRangePair(s, fallback, fallback)
	if a <= 0 && b <= 0 {
		return fallback
	}
	if b < a {
		b = a
	}
	return (a + b) / 2
}

func parseRangePair(s string, defA, defB int) (int, int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return defA, defB
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		a, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		b, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		if a <= 0 {
			a = defA
		}
		if b <= 0 {
			b = defB
		}
		return a, b
	}
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return defA, defB
	}
	return n, n
}

// FromSettings maps persisted settings into BuildInput (profile filled by caller).
func FromSettings(s *profiles.Settings) BuildInput {
	if s == nil {
		s = profiles.DefaultSettings()
	}
	return BuildInput{
		MixedPort:        s.MixedPort,
		ControllerURL:    s.ControllerURL,
		Secret:           s.Secret,
		Mode:             s.Mode,
		TUN:              s.TUN,
		TUNStack:         s.TUNStack,
		VPNInterface:     s.VPNInterface,
		AllowLAN:         s.AllowLAN,
		IPv6:             s.IPv6,
		LogLevel:         s.LogLevel,
		ProxyGroup:       s.ProxyGroup,
		BypassLAN:        s.BypassLAN,
		BypassGEOIP:      s.BypassGEOIP,
		RouteDirect:      s.RouteDirect,
		RouteBlock:       s.RouteBlock,
		RouteProxy:       s.RouteProxy,
		RouteWhitelist:   s.RouteWhitelist,
		AppRouteMode:     s.AppRouteMode,
		AppRouteList:     s.AppRouteList,
		DNSEnhancedMode:  s.DNSEnhancedMode,
		DNSNameservers:   profiles.SplitList(s.DNSNameservers),
		DNSFallbacks:     profiles.SplitList(s.DNSFallbacks),
		DNSFakeIPRange:   s.DNSFakeIPRange,
		Sniffer:          s.Sniffer,
		TCPConcurrent:    s.TCPConcurrent,
		UnifiedDelay:     s.UnifiedDelay,
		WARPEnabled:      s.WARPEnabled,
		WARPMode:         s.WARPMode,
		WARPPrivateKey:   s.WARPPrivateKey,
		WARPLocalAddress: s.WARPLocalAddress,
		WARPEndpoint:     s.WARPEndpoint,
		WARPPublicKey:    s.WARPPublicKey,
		WARPCleanIP:      s.WARPCleanIP,
		WARPPort:         s.WARPPort,
		WARPNoiseCount:   s.WARPNoiseCount,
		WARPNoiseMode:    s.WARPNoiseMode,
		WARPNoiseSize:    s.WARPNoiseSize,
		WARPNoiseDelay:   s.WARPNoiseDelay,
		URLTestURL:       defaults.ResolveURLTestURL(s.URLTestPreset, s.URLTestURL),
		URLTestSec:       s.URLTestIntervalSec,
	}
}
