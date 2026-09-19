package update

import (
	"strings"
	"testing"
)

func TestVersionMatches(t *testing.T) {
	cases := []struct {
		ver, asset, tag string
		want            bool
	}{
		{"", "MyInternetVPN-windows-amd64-abc1234.zip", "latest-main", false},
		{"dev", "MyInternetVPN-windows-amd64-abc1234.zip", "latest-main", false},
		{"abc1234", "MyInternetVPN-windows-amd64-abc1234.zip", "latest-main", true},
		{"abc123456789", "MyInternetVPN-windows-amd64-abc1234.zip", "latest-main", true},
		{"deadbeef", "MyInternetVPN-windows-amd64-abc1234.zip", "latest-main", false},
		{"latest-main", "MyInternetVPN-windows-amd64-abc1234.zip", "latest-main", true},
		{"1.2.3", "MyInternetVPN-1.2.3-windows.zip", "v1.2.3", true},
	}
	for _, tc := range cases {
		got := VersionMatches(tc.ver, tc.asset, tc.tag)
		if got != tc.want {
			t.Fatalf("VersionMatches(%q,%q,%q)=%v want %v", tc.ver, tc.asset, tc.tag, got, tc.want)
		}
	}
}

func TestBuildApplyScript(t *testing.T) {
	script := buildApplyScript(4242, `C:\Updates\staging`, `C:\Program Files\App`, `MyInternetVPN.exe`)
	for _, needle := range []string{
		"$pidToWait = 4242",
		"Wait-Process",
		"Copy-Item",
		"Start-Process",
		`C:\Updates\staging`,
		`MyInternetVPN.exe`,
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("script missing %q:\n%s", needle, script)
		}
	}
	if strings.Contains(script, "tasklist") || strings.Contains(script, "timeout /t") {
		t.Fatalf("script still uses cmd tasklist/timeout:\n%s", script)
	}
}

func TestParseBuildMarker(t *testing.T) {
	body := "Rolling signed build from `main`.\n\n- **Build:** `abc1234deadbeef`\n- **Assets:** zip\n"
	if got := parseBuildMarker(body); got != "abc1234deadbeef" {
		t.Fatalf("parseBuildMarker=%q", got)
	}
}

func TestFindWindowsZipPrefersFixedName(t *testing.T) {
	rel := Release{Assets: []ReleaseAsset{
		{Name: "MyInternetVPN-windows-amd64-oldhash.zip", UpdatedAt: "2026-01-02T00:00:00Z"},
		{Name: "MyInternetVPN-windows-amd64.zip", UpdatedAt: "2026-01-01T00:00:00Z"},
	}}
	c := &Checker{}
	asset, err := c.FindWindowsZip(rel)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Name != "MyInternetVPN-windows-amd64.zip" {
		t.Fatalf("got %s", asset.Name)
	}
}
