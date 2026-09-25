package config

import (
	"strings"
	"testing"

	"myinternetvpn/client/internal/profiles"
)

func TestParseAppListAndRules(t *testing.T) {
	apps := ParseAppList("AyuGram.exe\n# comment\nayugram.exe\n")
	if len(apps) != 1 {
		t.Fatalf("dedupe apps=%v", apps)
	}
	in := BuildInput{
		ProxyGroup:   "PROXY",
		AppRouteMode: AppRouteWhitelist,
		AppRouteList: "AyuGram.exe\nchrome.exe",
	}
	rules := AppendAppProcessRules(nil, in)
	if len(rules) != 2 {
		t.Fatalf("rules=%v", rules)
	}
	if rules[0] != "PROCESS-NAME,AyuGram.exe,PROXY" {
		t.Fatalf("got %s", rules[0])
	}
	in.AppRouteMode = AppRouteBlacklist
	rules = AppendAppProcessRules(nil, in)
	if rules[0] != "PROCESS-NAME,AyuGram.exe,DIRECT" {
		t.Fatalf("blacklist got %s", rules[0])
	}
}

func TestProcessPathRegexAndInstallDirs(t *testing.T) {
	rx := ProcessPathRegex(`C:\Users\me\AppData\Local\Programs\cursor`)
	if !strings.Contains(rx, "cursor") || !strings.Contains(rx, `(?i)`) {
		t.Fatalf("regex=%q", rx)
	}
	dirs := CollectAppInstallDirs("Cursor.exe", []AppPathHint{
		{Name: "Cursor.exe", Path: `C:\Users\me\AppData\Local\Programs\cursor\Cursor.exe`},
		{Name: "chrome.exe", Path: `C:\Program Files\Google\Chrome\Application\chrome.exe`},
	})
	if len(dirs) != 1 || !strings.Contains(strings.ToLower(dirs[0]), `programs\cursor`) {
		t.Fatalf("dirs=%v", dirs)
	}
	in := BuildInput{
		ProxyGroup:     "PROXY",
		AppRouteMode:   AppRouteWhitelist,
		AppRouteList:   "Cursor.exe",
		AppInstallDirs: dirs,
	}
	rules := AppendAppProcessRules(nil, in)
	joined := strings.Join(rules, "\n")
	if !strings.Contains(joined, "PROCESS-NAME,Cursor.exe,PROXY") {
		t.Fatalf("missing name rule: %v", rules)
	}
	if !strings.Contains(joined, "PROCESS-PATH-REGEX,") {
		t.Fatalf("missing path regex: %v", rules)
	}
}

func TestWhitelistProcessRulesBeforeDomains(t *testing.T) {
	in := BuildInput{
		ProxyGroup:     "PROXY",
		AppRouteMode:   AppRouteWhitelist,
		AppRouteList:   "Cursor.exe",
		AppInstallDirs: []string{`C:\Apps\cursor`},
		RouteDirect:    "cdn.cursor.sh",
		RouteProxy:     "api2.cursor.sh",
		BypassLAN:      false,
	}
	rules := buildRules(in)
	directIdx, procIdx, proxyIdx := -1, -1, -1
	for i, r := range rules {
		if strings.Contains(r, "cdn.cursor.sh") {
			directIdx = i
		}
		if strings.HasPrefix(r, "PROCESS-NAME,Cursor.exe,") || strings.HasPrefix(r, "PROCESS-PATH-REGEX,") {
			if procIdx < 0 {
				procIdx = i
			}
		}
		if strings.Contains(r, "api2.cursor.sh") {
			proxyIdx = i
		}
	}
	if directIdx < 0 || procIdx < 0 || proxyIdx < 0 {
		t.Fatalf("missing rules: %v", rules)
	}
	if !(directIdx < procIdx && procIdx < proxyIdx) {
		t.Fatalf("want DIRECT < PROCESS < PROXY order, got %v", rules)
	}
}

func TestBuildRulesAppWhitelist(t *testing.T) {
	in := BuildInput{
		ProxyGroup:   "PROXY",
		BypassLAN:    false,
		AppRouteMode: AppRouteWhitelist,
		AppRouteList: "AyuGram.exe",
		TUN:          true,
	}
	rules := buildRules(in)
	joined := strings.Join(rules, "\n")
	if !strings.Contains(joined, "PROCESS-NAME,AyuGram.exe,PROXY") {
		t.Fatalf("missing process rule: %v", rules)
	}
	if rules[len(rules)-1] != "MATCH,DIRECT" {
		t.Fatalf("want MATCH,DIRECT got %v", rules[len(rules)-1])
	}
}

func TestBuildRulesWhitelistKeepDirectEmptyList(t *testing.T) {
	in := BuildInput{
		ProxyGroup:             "PROXY",
		AppRouteMode:           AppRouteWhitelist,
		AppRouteList:           "",
		AppWhitelistKeepDirect: true,
		TUN:                    true,
	}
	rules := buildRules(in)
	if rules[len(rules)-1] != "MATCH,DIRECT" {
		t.Fatalf("emptied whitelist must stay MATCH,DIRECT, got %v", rules)
	}
	// Without KeepDirect, empty whitelist collapses to full tunnel.
	in.AppWhitelistKeepDirect = false
	rules = buildRules(in)
	if rules[len(rules)-1] != "MATCH,PROXY" {
		t.Fatalf("empty whitelist without keep → MATCH,PROXY, got %v", rules)
	}
}

func TestBuildAppWhitelistForcesRedirHost(t *testing.T) {
	raw, err := Build(BuildInput{
		Profile: profiles.Profile{
			Name: "Work",
			Proxies: []profiles.ProxyNode{
				{"name": "node-a", "type": "ss", "server": "a.example", "port": 443, "cipher": "aes-128-gcm", "password": "x"},
			},
		},
		TUN:             true,
		DNSEnhancedMode: "fake-ip",
		Sniffer:         true,
		AppRouteMode:    AppRouteWhitelist,
		AppRouteList:    "AyuGram.exe\nCursor.exe",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "enhanced-mode: redir-host") {
		t.Fatalf("expected redir-host for app whitelist:\n%s", text)
	}
	if !strings.Contains(text, "respect-rules: true") {
		t.Fatalf("expected respect-rules:\n%s", text)
	}
	if !strings.Contains(text, "proxy-server-nameserver:") {
		t.Fatalf("expected proxy-server-nameserver with respect-rules:\n%s", text)
	}
	if !strings.Contains(text, "find-process-mode: always") {
		t.Fatalf("expected find-process-mode:\n%s", text)
	}
	if !strings.Contains(text, "direct-nameserver:") || !strings.Contains(text, "system") {
		t.Fatalf("expected direct-nameserver system:\n%s", text)
	}
	if !strings.Contains(text, "endpoint-independent-nat: true") {
		t.Fatalf("expected endpoint-independent-nat for whitelist:\n%s", text)
	}
	if strings.Contains(text, "sniffer:") {
		t.Fatalf("sniffer should be off for app split-tunnel without site rules:\n%s", text)
	}
}

func TestBuildAppRoutingKeepsSnifferForSiteRules(t *testing.T) {
	raw, err := Build(BuildInput{
		Profile: profiles.Profile{
			Name: "Work",
			Proxies: []profiles.ProxyNode{
				{"name": "node-a", "type": "ss", "server": "a.example", "port": 443, "cipher": "aes-128-gcm", "password": "x"},
			},
		},
		TUN:             true,
		DNSEnhancedMode: "fake-ip",
		Sniffer:         true,
		AppRouteMode:    AppRouteWhitelist,
		AppRouteList:    "Telegram.exe",
		RouteProxy:      "api2.cursor.sh\n*.authentication.cursor.sh",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "sniffer:") {
		t.Fatalf("expected sniffer when site rules + app routing:\n%s", text)
	}
	if !strings.Contains(text, "force-dns-mapping: true") {
		t.Fatalf("expected force-dns-mapping:\n%s", text)
	}
	if strings.Contains(text, "override-destination: true") {
		t.Fatalf("override-destination must stay false (Cloudflare Access):\n%s", text)
	}
	if !strings.Contains(text, "cloudflareaccess.com") {
		t.Fatalf("expected cloudflareaccess skip-domain:\n%s", text)
	}
	if !strings.Contains(text, "DOMAIN-SUFFIX,api2.cursor.sh") {
		t.Fatalf("expected cursor domain rule:\n%s", text)
	}
	if !strings.Contains(text, "PROCESS-NAME,Telegram.exe,") {
		t.Fatalf("expected app process rule:\n%s", text)
	}
}

func TestProcessPrefixInCustomRules(t *testing.T) {
	rules := ParseCustomRules("process:steam.exe\napp:discord.exe", "DIRECT")
	if len(rules) != 2 {
		t.Fatalf("%v", rules)
	}
	if rules[0] != "PROCESS-NAME,steam.exe,DIRECT" {
		t.Fatalf("%s", rules[0])
	}
}
