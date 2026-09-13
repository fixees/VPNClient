package config

import (
	"fmt"

	"myinternetvpn/client/internal/profiles"

	"gopkg.in/yaml.v3"
)

// BuildInput drives runtime mihomo config generation.
type BuildInput struct {
	Profile       profiles.Profile
	MixedPort     int
	ControllerURL string
	Secret        string
	Mode          string
	TUN           bool
	LogLevel      string
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
		in.MixedPort = 7890
	}
	if in.ControllerURL == "" {
		in.ControllerURL = "127.0.0.1:9090"
	}
	if in.Mode == "" {
		in.Mode = "rule"
	}
	if in.LogLevel == "" {
		in.LogLevel = "info"
	}

	proxyNames := make([]string, 0, len(in.Profile.Proxies))
	proxies := make([]map[string]any, 0, len(in.Profile.Proxies))
	for i, p := range in.Profile.Proxies {
		name, _ := p["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("proxy at index %d missing name", i)
		}
		proxyNames = append(proxyNames, name)
		proxies = append(proxies, map[string]any(p))
	}

	doc := map[string]any{
		"mixed-port":          in.MixedPort,
		"allow-lan":           false,
		"mode":                in.Mode,
		"log-level":           in.LogLevel,
		"ipv6":                false,
		"external-controller": in.ControllerURL,
		"secret":              in.Secret,
		"dns": map[string]any{
			"enable":       true,
			"enhanced-mode": "fake-ip",
			"nameserver":   []string{"8.8.8.8", "1.1.1.1"},
			"fallback":     []string{"tls://1.1.1.1:853"},
		},
		"proxies": proxies,
		"proxy-groups": []map[string]any{
			{
				"name":    "PROXY",
				"type":    "select",
				"proxies": append([]string{"AUTO"}, proxyNames...),
			},
			{
				"name":     "AUTO",
				"type":     "url-test",
				"proxies":  proxyNames,
				"url":      "https://www.gstatic.com/generate_204",
				"interval": 300,
			},
		},
		"rules": []string{
			"GEOIP,CN,DIRECT",
			"MATCH,PROXY",
		},
	}

	if in.TUN {
		doc["tun"] = map[string]any{
			"enable":               true,
			"stack":                "system",
			"auto-route":           true,
			"auto-detect-interface": true,
			"dns-hijack":           []string{"any:53"},
		}
	}

	// Prefer local geo databases when present in workdir (-d).
	doc["geodata-mode"] = true

	return yaml.Marshal(doc)
}
