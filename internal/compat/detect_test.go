package compat_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"myinternetvpn/client/internal/compat"
	"myinternetvpn/client/internal/config"
	"myinternetvpn/client/internal/winutil"
)

func TestDetectZapretOverlapWhitelist(t *testing.T) {
	cat := compat.Catalog{Tools: []compat.Tool{{
		ID:    "zapret",
		Title: "zapret",
		Match: compat.ToolMatch{
			ProcessNames: []string{"winws.exe"},
		},
		Handles: compat.ToolHandles{
			AppNames:       []string{"Discord.exe", "DiscordPTB.exe"},
			DomainKeywords: []string{"discord", "youtube"},
		},
		WhenOverlap: compat.OverlapPolicy{
			Severity: "warn_and_prefer_direct",
			Message:  "zapret owns discord",
		},
	}}}
	apps := []winutil.RunningApp{{Name: "winws.exe", Path: `C:\tools\zapret\winws.exe`}}
	findings := compat.Detect(cat, apps, compat.RouteContext{
		Mode:    "whitelist",
		Apps:    []string{"Discord.exe", "chrome.exe"},
		Domains: []string{"*.discord.com", "example.com"},
	})
	if len(findings) != 1 {
		t.Fatalf("findings=%v", findings)
	}
	if !findings[0].PreferDirect {
		t.Fatal("expected prefer direct")
	}
	if len(findings[0].OverlapApps) != 1 || !strings.EqualFold(findings[0].OverlapApps[0], "Discord.exe") {
		t.Fatalf("overlap=%v", findings[0].OverlapApps)
	}
	if len(findings[0].OverlapDomains) != 1 || !strings.Contains(findings[0].OverlapDomains[0], "discord") {
		t.Fatalf("domains=%v", findings[0].OverlapDomains)
	}
	findings = compat.Detect(cat, apps, compat.RouteContext{
		Mode: "blacklist",
		Apps: []string{"Discord.exe"},
	})
	if len(findings) != 1 || len(findings[0].OverlapApps) != 0 {
		t.Fatalf("blacklist overlap should be empty: %#v", findings[0])
	}
}

func TestRemoveAndMergeAppList(t *testing.T) {
	list := "Telegram.exe\nDiscord.exe\nchrome.exe"
	got := compat.RemoveAppsFromList(list, []string{"discord.exe"})
	if strings.Contains(strings.ToLower(got), "discord") {
		t.Fatalf("still has discord: %q", got)
	}
	merged := compat.MergeAppList(got, []string{"Discord.exe", "Update.exe"})
	if !strings.Contains(merged, "Update.exe") || !strings.Contains(merged, "Discord.exe") {
		t.Fatalf("merge failed: %q", merged)
	}
	proxy := "*.discord.com\nexample.com"
	stripped := compat.RemoveLinesFromList(proxy, []string{"*.discord.com"})
	if strings.Contains(stripped, "discord") || !strings.Contains(stripped, "example.com") {
		t.Fatalf("domain strip failed: %q", stripped)
	}
}

func TestEmbeddedCatalogLoads(t *testing.T) {
	if err := compat.InitEmbeddedCatalog(); err != nil {
		t.Fatal(err)
	}
	cat, ok := compat.DefaultCatalog()
	if !ok || len(cat.Tools) == 0 {
		t.Fatal("empty catalog")
	}
	found := false
	for _, tool := range cat.Tools {
		if tool.ID == "zapret" {
			found = true
			if len(tool.Match.ProcessNames) == 0 {
				t.Fatal("zapret has no process names")
			}
			if len(tool.Handles.DomainKeywords) == 0 {
				t.Fatal("zapret domain keywords missing")
			}
		}
	}
	if !found {
		t.Fatal("zapret missing from catalog")
	}
}

func TestExpandAppFamilySameDirOnly(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Dir(dir)
	_ = os.WriteFile(filepath.Join(dir, "Discord.exe"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "DiscordCrashHandler.exe"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "Update.exe"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "unrelated-tool.exe"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(parent, "OtherProduct.exe"), []byte("x"), 0o644)

	got := compat.ExpandAppFamily(filepath.Join(dir, "Discord.exe"))
	joined := strings.ToLower(strings.Join(got, ","))
	if !strings.Contains(joined, "discord.exe") {
		t.Fatalf("missing main: %v", got)
	}
	if strings.Contains(joined, "otherproduct") {
		t.Fatalf("parent leaked: %v", got)
	}
	if strings.Contains(joined, "unrelated-tool") {
		t.Fatalf("unrelated sibling: %v", got)
	}
}

func TestWhitelistKeepDirectWhenEmptied(t *testing.T) {
	in := config.BuildInput{
		AppRouteMode:           config.AppRouteWhitelist,
		AppRouteList:           "",
		AppWhitelistKeepDirect: true,
		ProxyGroup:             "PROXY",
	}
	if !config.AppWhitelistActive(in) {
		t.Fatal("expected whitelist active with KeepDirect")
	}
	if !config.AppRoutingEnabled(in) {
		t.Fatal("expected app routing enabled")
	}
}
