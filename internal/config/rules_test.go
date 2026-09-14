package config

import "testing"

func TestParseCustomRules(t *testing.T) {
	text := `
# comment
example.com
*.google.com
domain:exact.foo
suffix:bar.com
keyword:ads
regexp:^tracker\.
/ads\..*/
geoip:CN
geosite:category-ads-all
10.0.0.0/8
1.2.3.4
ip:8.8.8.8/32
2001:db8::/32
`
	got := ParseCustomRules(text, "DIRECT")
	want := []string{
		"DOMAIN-SUFFIX,example.com,DIRECT",
		"DOMAIN-SUFFIX,google.com,DIRECT",
		"DOMAIN,exact.foo,DIRECT",
		"DOMAIN-SUFFIX,bar.com,DIRECT",
		"DOMAIN-KEYWORD,ads,DIRECT",
		"DOMAIN-REGEX,^tracker\\.,DIRECT",
		"DOMAIN-REGEX,ads\\..*,DIRECT",
		"GEOIP,CN,DIRECT",
		"GEOSITE,category-ads-all,DIRECT",
		"IP-CIDR,10.0.0.0/8,DIRECT,no-resolve",
		"IP-CIDR,1.2.3.4/32,DIRECT,no-resolve",
		"IP-CIDR,8.8.8.8/32,DIRECT,no-resolve",
		"IP-CIDR6,2001:db8::/32,DIRECT,no-resolve",
	}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d\ngot=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestParseCustomRulesReject(t *testing.T) {
	got := ParseCustomRules("ads.example\nregexp:(?i)tracker", "REJECT")
	if len(got) != 2 || !contains(got, "DOMAIN-SUFFIX,ads.example,REJECT") {
		t.Fatalf("%v", got)
	}
}

func contains(list []string, needle string) bool {
	for _, s := range list {
		if s == needle {
			return true
		}
	}
	return false
}
