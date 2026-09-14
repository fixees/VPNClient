package tests

import (
	"context"
	"testing"
	"time"

	"myinternetvpn/client/internal/defaults"
	"myinternetvpn/client/internal/latency"
	"myinternetvpn/client/internal/profiles"
)

func TestProxyEndpoint(t *testing.T) {
	host, port, err := latency.ProxyEndpoint(profiles.ProxyNode{
		"name": "n", "server": "1.2.3.4", "port": 443,
	})
	if err != nil || host != "1.2.3.4" || port != 443 {
		t.Fatalf("got %s:%d err=%v", host, port, err)
	}
}

func TestProbeDirectHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ms, err := latency.Probe(ctx, latency.Options{
		Method:  defaults.PingHTTPGet,
		TestURL: "https://www.gstatic.com/generate_204",
		Timeout: 5 * time.Second,
		Proxy:   profiles.ProxyNode{"name": "x"},
	})
	if err != nil {
		t.Skipf("network unavailable: %v", err)
	}
	if ms <= 0 {
		t.Fatalf("delay=%d", ms)
	}
}

func TestResolveURLPresets(t *testing.T) {
	if got := defaults.ResolveURLTestURL("cloudflare", ""); got == "" || got == defaults.URLTestURL {
		// cloudflare preset should differ from default gstatic
		if got != defaults.URLTestPresets["cloudflare"] {
			t.Fatalf("cloudflare preset = %q", got)
		}
	}
	if got := defaults.ResolveURLTestURL("custom", "https://example.com/204"); got != "https://example.com/204" {
		t.Fatalf("custom = %q", got)
	}
}
