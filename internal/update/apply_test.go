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

func TestPsQuote(t *testing.T) {
	if got := psQuote(`C:\O'Brien\app`); got != `'C:\O''Brien\app'` {
		t.Fatalf("psQuote: %q", got)
	}
}
