package netinfo

import (
	"strings"
	"testing"
)

func TestMaskIP(t *testing.T) {
	if got := MaskIP("185.42.10.99"); got != "185.***.***.99" {
		t.Fatalf("ipv4: %q", got)
	}
	if got := MaskIP(""); got != "-" {
		t.Fatalf("empty: %q", got)
	}
	got := MaskIP("2001:db8:85a3::8a2e:370:7334")
	if !strings.Contains(got, "****") || !strings.HasPrefix(got, "2001:") {
		t.Fatalf("ipv6: %q", got)
	}
}
