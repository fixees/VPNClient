package config

import (
	"net"
	"strings"

	"myinternetvpn/client/internal/defaults"
)

// ParseCustomRules turns multiline user lists into Clash/mihomo rule strings.
// Supported lines (one per line, # comments):
//
//	example.com              → DOMAIN-SUFFIX
//	*.google.com             → DOMAIN-SUFFIX,google.com
//	domain:foo.com           → DOMAIN,foo.com
//	suffix:foo.com           → DOMAIN-SUFFIX,foo.com
//	keyword:ads              → DOMAIN-KEYWORD,ads
//	regexp:^ads\..*          → DOMAIN-REGEX,^ads\..*
//	/ads\..*/                → DOMAIN-REGEX,ads\..*
//	1.2.3.4  / 10.0.0.0/8    → IP-CIDR
//	2001:db8::/32            → IP-CIDR6
//	geoip:CN                 → GEOIP,CN
//	geosite:category-ads     → GEOSITE,category-ads
//	ip:1.2.3.0/24            → IP-CIDR
func ParseCustomRules(text, action string) []string {
	action = strings.TrimSpace(action)
	if action == "" {
		action = "DIRECT"
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule, ok := lineToRule(line, action)
		if !ok || rule == "" {
			continue
		}
		if _, dup := seen[rule]; dup {
			continue
		}
		seen[rule] = struct{}{}
		out = append(out, rule)
	}
	return out
}

func lineToRule(line, action string) (string, bool) {
	lower := strings.ToLower(line)

	switch {
	case strings.HasPrefix(lower, "process:"), strings.HasPrefix(lower, "app:"):
		idx := strings.Index(line, ":")
		v := NormalizeProcessName(line[idx+1:])
		if v == "" {
			return "", false
		}
		if strings.ContainsAny(v, `/\`) {
			return "PROCESS-PATH," + v + "," + action, true
		}
		return "PROCESS-NAME," + v + "," + action, true
	case strings.HasPrefix(lower, "geoip:"):
		code := strings.ToUpper(strings.TrimSpace(line[len("geoip:"):]))
		if code == "" {
			return "", false
		}
		return "GEOIP," + code + "," + action, true
	case strings.HasPrefix(lower, "geosite:"):
		name := strings.TrimSpace(line[len("geosite:"):])
		if name == "" {
			return "", false
		}
		return "GEOSITE," + name + "," + action, true
	case strings.HasPrefix(lower, "domain:"):
		v := strings.TrimSpace(line[len("domain:"):])
		if v == "" {
			return "", false
		}
		return "DOMAIN," + v + "," + action, true
	case strings.HasPrefix(lower, "suffix:"):
		v := strings.TrimPrefix(strings.TrimSpace(line[len("suffix:"):]), "*.")
		if v == "" {
			return "", false
		}
		return "DOMAIN-SUFFIX," + v + "," + action, true
	case strings.HasPrefix(lower, "keyword:"):
		v := strings.TrimSpace(line[len("keyword:"):])
		if v == "" {
			return "", false
		}
		return "DOMAIN-KEYWORD," + v + "," + action, true
	case strings.HasPrefix(lower, "regexp:"), strings.HasPrefix(lower, "regex:"):
		idx := strings.Index(line, ":")
		v := strings.TrimSpace(line[idx+1:])
		if v == "" {
			return "", false
		}
		return "DOMAIN-REGEX," + v + "," + action, true
	case strings.HasPrefix(lower, "ip:"):
		v := strings.TrimSpace(line[len("ip:"):])
		return ipRule(v, action)
	}

	// /pattern/ → DOMAIN-REGEX
	if len(line) >= 2 && strings.HasPrefix(line, "/") && strings.HasSuffix(line, "/") {
		pat := line[1 : len(line)-1]
		if pat == "" {
			return "", false
		}
		return "DOMAIN-REGEX," + pat + "," + action, true
	}

	if rule, ok := ipRule(line, action); ok {
		return rule, true
	}

	// *.example.com → DOMAIN-SUFFIX
	if strings.HasPrefix(line, "*.") {
		v := strings.TrimPrefix(line, "*.")
		if v == "" {
			return "", false
		}
		return "DOMAIN-SUFFIX," + v + "," + action, true
	}

	// Bare domain → DOMAIN-SUFFIX (covers subdomains too — useful for bypass lists)
	if isDomainLike(line) {
		return "DOMAIN-SUFFIX," + strings.TrimPrefix(line, ".") + "," + action, true
	}
	return "", false
}

func ipRule(v, action string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	noResolve := ",no-resolve"
	if strings.Contains(v, "/") {
		ip, _, err := net.ParseCIDR(v)
		if err != nil {
			return "", false
		}
		if ip.To4() != nil {
			return "IP-CIDR," + v + "," + action + noResolve, true
		}
		return "IP-CIDR6," + v + "," + action + noResolve, true
	}
	ip := net.ParseIP(v)
	if ip == nil {
		return "", false
	}
	if ip.To4() != nil {
		return "IP-CIDR," + v + "/32," + action + noResolve, true
	}
	return "IP-CIDR6," + v + "/128," + action + noResolve, true
}

func isDomainLike(s string) bool {
	if s == "" || strings.ContainsAny(s, " /\\") {
		return false
	}
	if net.ParseIP(s) != nil {
		return false
	}
	// allow unicode domains / punycode / dots
	if !strings.Contains(s, ".") && !strings.Contains(s, "xn--") {
		// single-label like "localhost" still ok as DOMAIN-SUFFIX
		for _, r := range s {
			if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r > 127) {
				return false
			}
		}
		return len(s) > 0
	}
	return true
}

// AppendCustomRules inserts user rules before MATCH (after LAN/GEO helpers caller order).
func AppendCustomRules(rules []string, in BuildInput) []string {
	proxyTarget := in.ProxyGroup
	if proxyTarget == "" {
		proxyTarget = defaults.ProxyGroup
	}
	// Block first, then direct bypass, then force-proxy.
	rules = append(rules, ParseCustomRules(in.RouteBlock, "REJECT")...)
	rules = append(rules, ParseCustomRules(in.RouteDirect, "DIRECT")...)
	rules = append(rules, ParseCustomRules(in.RouteProxy, proxyTarget)...)
	return rules
}
