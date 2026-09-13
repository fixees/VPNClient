package tests

import (
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
		MixedPort:     17890,
		ControllerURL: "127.0.0.1:19090",
		Secret:        "test-secret",
		Mode:          "rule",
		TUN:           true,
		LogLevel:      "debug",
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
	if doc["external-controller"] != "127.0.0.1:19090" {
		t.Fatalf("controller = %v", doc["external-controller"])
	}
	if doc["secret"] != "test-secret" {
		t.Fatalf("secret = %v", doc["secret"])
	}
	tun, ok := doc["tun"].(map[string]any)
	if !ok || tun["enable"] != true {
		t.Fatalf("expected tun enabled, got %#v", doc["tun"])
	}
	proxies, ok := doc["proxies"].([]any)
	if !ok || len(proxies) != 2 {
		t.Fatalf("proxies = %#v", doc["proxies"])
	}
	text := string(raw)
	if !strings.Contains(text, "node-a") || !strings.Contains(text, "PROXY") {
		t.Fatalf("unexpected yaml content:\n%s", text)
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
		TUN: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, needle := range []string{"de-1", "nl-1", "type: select", "type: url-test", "MATCH,PROXY"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %q in:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "tun:") {
		t.Fatal("tun should be omitted when disabled")
	}
}
