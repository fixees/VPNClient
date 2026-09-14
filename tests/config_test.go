package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"myinternetvpn/client/internal/config"
	"myinternetvpn/client/internal/profiles"

	"gopkg.in/yaml.v3"
)

func TestConfigBuildProducesValidYAML(t *testing.T) {
	raw, err := config.Build(config.BuildInput{
		Profile: profiles.Profile{
			Name: "Work",
			Proxies: []profiles.ProxyNode{
				{"name": "node-a", "type": "ss", "server": "a.example", "port": 443, "cipher": "aes-128-gcm", "password": "x"},
				{"name": "node-b", "type": "vmess", "server": "b.example", "port": 443, "uuid": "u", "alterId": 0, "cipher": "auto"},
			},
		},
		MixedPort:       17890,
		ControllerURL:   "127.0.0.1:19090",
		Secret:          "test-secret",
		Mode:            "rule",
		TUN:             true,
		TUNStack:        "gvisor",
		AllowLAN:        true,
		IPv6:            true,
		LogLevel:        "debug",
		BypassLAN:       true,
		BypassGEOIP:     "CN",
		DNSEnhancedMode: "fake-ip",
		Sniffer:         true,
		TCPConcurrent:   true,
		UnifiedDelay:    true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["mixed-port"] != 17890 {
		t.Fatalf("mixed-port = %v", doc["mixed-port"])
	}
	if doc["allow-lan"] != true {
		t.Fatalf("allow-lan = %v", doc["allow-lan"])
	}
	if doc["ipv6"] != true {
		t.Fatalf("ipv6 = %v", doc["ipv6"])
	}
	if doc["external-controller"] != "127.0.0.1:19090" {
		t.Fatalf("controller = %v", doc["external-controller"])
	}
	if doc["secret"] != "test-secret" {
		t.Fatalf("secret = %v", doc["secret"])
	}
	tun, ok := doc["tun"].(map[string]any)
	if !ok || tun["enable"] != true || tun["stack"] != "gvisor" {
		t.Fatalf("expected tun enabled/gvisor, got %#v", doc["tun"])
	}
	if tun["device-name"] != "Meta" {
		t.Fatalf("expected device-name Meta, got %#v", tun["device-name"])
	}
	dns, ok := doc["dns"].(map[string]any)
	if !ok || dns["enhanced-mode"] != "fake-ip" {
		t.Fatalf("dns = %#v", doc["dns"])
	}
	if _, ok := doc["sniffer"].(map[string]any); !ok {
		t.Fatalf("expected sniffer, got %#v", doc["sniffer"])
	}
	proxies, ok := doc["proxies"].([]any)
	if !ok || len(proxies) != 2 {
		t.Fatalf("proxies = %#v", doc["proxies"])
	}
	text := string(raw)
	if !strings.Contains(text, "node-a") || !strings.Contains(text, "PROXY") {
		t.Fatalf("unexpected yaml content:\n%s", text)
	}
	if !strings.Contains(text, "GEOIP,CN,DIRECT") {
		t.Fatalf("missing GEOIP rule:\n%s", text)
	}
	if !strings.Contains(text, "IP-CIDR,192.168.0.0/16,DIRECT") {
		t.Fatalf("missing LAN bypass:\n%s", text)
	}
}

func TestConfigBuildValidation(t *testing.T) {
	if _, err := config.Build(config.BuildInput{}); err == nil {
		t.Fatal("expected error for empty profile")
	}
	if _, err := config.Build(config.BuildInput{
		Profile: profiles.Profile{Name: "x", Proxies: []profiles.ProxyNode{{"type": "ss"}}},
	}); err == nil {
		t.Fatal("expected missing proxy name error")
	}
}

func TestConfigBuildUsesAllProxyNamesInGroups(t *testing.T) {
	raw, err := config.Build(config.BuildInput{
		Profile: profiles.Profile{
			Name: "EU",
			Proxies: []profiles.ProxyNode{
				{"name": "de-1", "type": "ss", "server": "de.example", "port": 443, "cipher": "aes-128-gcm", "password": "p"},
				{"name": "nl-1", "type": "trojan", "server": "nl.example", "port": 443, "password": "p"},
			},
		},
		TUN:         false,
		BypassGEOIP: "CN",
		RouteBlock:  "ads.bad\nregexp:^tracker\\.",
		RouteDirect: "intranet.corp\n10.20.0.0/16",
		RouteProxy:  "need-vpn.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, needle := range []string{
		"de-1", "nl-1", "type: select", "type: url-test", "MATCH,PROXY", "GEOIP,CN,DIRECT",
		"DOMAIN-SUFFIX,ads.bad,REJECT",
		"DOMAIN-REGEX,^tracker\\.,REJECT",
		"DOMAIN-SUFFIX,intranet.corp,DIRECT",
		"IP-CIDR,10.20.0.0/16,DIRECT,no-resolve",
		"DOMAIN-SUFFIX,need-vpn.example,PROXY",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %q in:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "tun:") {
		t.Fatal("tun should be omitted when disabled")
	}
}

func TestConfigBuildWhitelistMode(t *testing.T) {
	raw, err := config.Build(config.BuildInput{
		Profile: profiles.Profile{
			Name: "WL",
			Proxies: []profiles.ProxyNode{
				{"name": "n1", "type": "ss", "server": "a.example", "port": 443, "cipher": "aes-128-gcm", "password": "p"},
			},
		},
		BypassGEOIP:    "CN",
		BypassLAN:      true,
		RouteDirect:    "ok.direct",
		RouteProxy:     "ok.vpn",
		RouteWhitelist: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "MATCH,REJECT") {
		t.Fatalf("expected MATCH,REJECT:\n%s", text)
	}
	if strings.Contains(text, "GEOIP,CN,DIRECT") {
		t.Fatal("GEOIP bypass must be off in whitelist mode")
	}
	if !strings.Contains(text, "DOMAIN-SUFFIX,ok.direct,DIRECT") || !strings.Contains(text, "DOMAIN-SUFFIX,ok.vpn,PROXY") {
		t.Fatalf("allow rules missing:\n%s", text)
	}
	if !strings.Contains(text, "IP-CIDR,192.168.0.0/16,DIRECT") {
		t.Fatal("LAN bypass should remain")
	}
}

func TestConfigBuildWARP(t *testing.T) {
	_, err := config.Build(config.BuildInput{
		Profile: profiles.Profile{
			Name:    "W",
			Proxies: []profiles.ProxyNode{{"name": "n1", "type": "ss", "server": "x", "port": 1, "cipher": "aes-128-gcm", "password": "p"}},
		},
		WARPEnabled: true,
	})
	if err == nil {
		t.Fatal("expected WARP key error")
	}

	raw, err := config.Build(config.BuildInput{
		Profile: profiles.Profile{
			Name:    "W",
			Proxies: []profiles.ProxyNode{{"name": "n1", "type": "ss", "server": "x", "port": 1, "cipher": "aes-128-gcm", "password": "p"}},
		},
		WARPEnabled:    true,
		WARPPrivateKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		WARPEndpoint:   "engage.cloudflareclient.com:2408",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "type: wireguard") || !strings.Contains(text, "name: WARP") {
		t.Fatalf("WARP proxy missing:\n%s", text)
	}
	if !strings.Contains(text, "dialer-proxy: PROXY") {
		t.Fatalf("expected WARP dialer-proxy through PROXY:\n%s", text)
	}
	if !strings.Contains(text, "MATCH,WARP") {
		t.Fatalf("expected MATCH,WARP exit:\n%s", text)
	}
	if strings.Contains(text, "proxies:\n    - AUTO\n    - n1\n    - WARP") || strings.Count(text, "- WARP") > 1 && strings.Contains(text, "type: select") {
		// WARP must not be a selectable leaf in PROXY group list of names after AUTO
	}
	// PROXY select group should list AUTO + n1 only (not WARP as choice).
	if strings.Contains(text, "type: select") {
		idx := strings.Index(text, "type: select")
		chunk := text[idx:]
		if end := strings.Index(chunk, "type: url-test"); end > 0 {
			chunk = chunk[:end]
		}
		if strings.Contains(chunk, "- WARP") {
			t.Fatalf("WARP must not appear in select group:\n%s", chunk)
		}
	}
}

func TestConfigFromSettings(t *testing.T) {
	s := profiles.DefaultSettings()
	s.AllowLAN = true
	s.DNSNameservers = "1.1.1.1, 8.8.8.8"
	in := config.FromSettings(s)
	if !in.AllowLAN || len(in.DNSNameservers) != 2 {
		t.Fatalf("FromSettings = %#v", in)
	}
}

func TestMihomoAcceptsGeneratedConfig(t *testing.T) {
	core := filepath.Join("..", "resources", "core", "mihomo.exe")
	if _, err := os.Stat(core); err != nil {
		t.Skip("mihomo.exe not available")
	}
	s := profiles.DefaultSettings()
	in := config.FromSettings(s)
	in.Profile = profiles.Profile{
		Name: "validate",
		Proxies: []profiles.ProxyNode{
			{"name": "node-a", "type": "ss", "server": "1.1.1.1", "port": 443, "cipher": "aes-128-gcm", "password": "x"},
		},
	}
	raw, err := config.Build(in)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	// Copy geo assets if present so geodata-mode does not fail hard.
	for _, name := range []string{"geoip.metadb", "geosite.dat"} {
		src := filepath.Join("..", "resources", "core", name)
		if b, err := os.ReadFile(src); err == nil {
			_ = os.WriteFile(filepath.Join(dir, name), b, 0o644)
		}
	}
	cmd := exec.Command(core, "-t", "-f", cfgPath, "-d", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mihomo -t failed: %v\n%s\nconfig:\n%s", err, out, raw)
	}
}
