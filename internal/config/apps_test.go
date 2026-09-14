package config

import (
	"strings"
	"testing"
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

func TestProcessPrefixInCustomRules(t *testing.T) {
	rules := ParseCustomRules("process:steam.exe\napp:discord.exe", "DIRECT")
	if len(rules) != 2 {
		t.Fatalf("%v", rules)
	}
	if rules[0] != "PROCESS-NAME,steam.exe,DIRECT" {
		t.Fatalf("%s", rules[0])
	}
}
