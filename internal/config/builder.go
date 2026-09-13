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
	AllowLAN         bool
	IPv6             bool
	LogLevel         string
	ProxyGroup       string
	URLTestURL       string
	URLTestSec       int
	BypassLAN        bool
	BypassGEOIP      string
	DNSEnhancedMode  string
	DNSNameservers   []string
	DNSFallbacks     []string
	DNSFakeIPRange   string
	Sniffer          bool
	TCPConcurrent    bool
	UnifiedDelay     bool
	WARPEnabled      bool
	WARPPrivateKey   string
	WARPLocalAddress string
	WARPEndpoint     string
	WARPPublicKey    string
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

	proxyNames := make([]string, 0, len(in.Profile.Proxies)+1)
	proxies := make([]map[string]any, 0, len(in.Profile.Proxies)+1)
	for i, p := range in.Profile.Proxies {
		name, _ := p["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("proxy at index %d missing name", i)
		}
		proxyNames = append(proxyNames, name)
		proxies = append(proxies, map[string]any(p))
	}

	if in.WARPEnabled {
		warp, err := buildWARPProxy(in)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, warp)
		proxyNames = append(proxyNames, defaults.WARPProxyName)
	}

	selectProxies := append([]string{defaults.AutoGroup}, proxyNames...)
	rules := buildRules(in)

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
		"dns": map[string]any{
			"enable":        true,
			"enhanced-mode": in.DNSEnhancedMode,
			"fake-ip-range": in.DNSFakeIPRange,
			"nameserver":    in.DNSNameservers,
			"fallback":      in.DNSFallbacks,
			"ipv6":          in.IPv6,
		},
		"proxies": proxies,
		"proxy-groups": []map[string]any{
			{
				"name":    in.ProxyGroup,
				"type":    "select",
				"proxies": selectProxies,
			},
			{
				"name":     defaults.AutoGroup,
				"type":     "url-test",
				"proxies":  proxyNames,
				"url":      in.URLTestURL,
				"interval": in.URLTestSec,
			},
		},
		"rules": rules,
	}

	if in.TUN {
		doc["tun"] = map[string]any{
			"enable":                true,
			"stack":                 in.TUNStack,
			"auto-route":            true,
			"auto-detect-interface": true,
			"dns-hijack":            []string{"any:53"},
		}
	}

	if in.Sniffer {
		doc["sniffer"] = map[string]any{
			"enable":               true,
			"parse-pure-ip":        true,
			"override-destination": true,
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
	if geo := strings.ToUpper(strings.TrimSpace(in.BypassGEOIP)); geo != "" && geo != "OFF" && geo != "NONE" {
		rules = append(rules, "GEOIP,"+geo+",DIRECT")
	}
	rules = append(rules, "MATCH,"+in.ProxyGroup)
	return rules
}

func buildWARPProxy(in BuildInput) (map[string]any, error) {
	key := strings.TrimSpace(in.WARPPrivateKey)
	if key == "" {
		return nil, fmt.Errorf("WARP включён, но не задан private key")
	}
	endpoint := strings.TrimSpace(in.WARPEndpoint)
	if endpoint == "" {
		endpoint = defaults.WARPEndpoint
	}
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		// allow host without port
		host = endpoint
		portStr = "2408"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("invalid WARP endpoint port")
	}
	pub := strings.TrimSpace(in.WARPPublicKey)
	if pub == "" {
		pub = defaults.WARPPublicKey
	}
	local := strings.TrimSpace(in.WARPLocalAddress)
	if local == "" {
		local = defaults.WARPLocalAddress
	}
	return map[string]any{
		"name":        defaults.WARPProxyName,
		"type":        "wireguard",
		"server":      host,
		"port":        port,
		"ip":          strings.TrimSuffix(local, "/32"),
		"private-key": key,
		"public-key":  pub,
		"udp":         true,
		"mtu":         1280,
	}, nil
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
		AllowLAN:         s.AllowLAN,
		IPv6:             s.IPv6,
		LogLevel:         s.LogLevel,
		ProxyGroup:       s.ProxyGroup,
		BypassLAN:        s.BypassLAN,
		BypassGEOIP:      s.BypassGEOIP,
		DNSEnhancedMode:  s.DNSEnhancedMode,
		DNSNameservers:   profiles.SplitList(s.DNSNameservers),
		DNSFallbacks:     profiles.SplitList(s.DNSFallbacks),
		DNSFakeIPRange:   s.DNSFakeIPRange,
		Sniffer:          s.Sniffer,
		TCPConcurrent:    s.TCPConcurrent,
		UnifiedDelay:     s.UnifiedDelay,
		WARPEnabled:      s.WARPEnabled,
		WARPPrivateKey:   s.WARPPrivateKey,
		WARPLocalAddress: s.WARPLocalAddress,
		WARPEndpoint:     s.WARPEndpoint,
		WARPPublicKey:    s.WARPPublicKey,
	}
}
