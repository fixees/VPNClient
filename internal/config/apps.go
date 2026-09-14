package config

import (
	"path/filepath"
	"strings"

	"myinternetvpn/client/internal/defaults"
)

const (
	AppRouteOff       = "off"
	AppRouteWhitelist = "whitelist" // only listed apps → VPN; rest DIRECT
	AppRouteBlacklist = "blacklist" // listed apps → DIRECT; rest VPN
)

// NormalizeAppRouteMode returns off|whitelist|blacklist.
func NormalizeAppRouteMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AppRouteWhitelist, "allow", "only":
		return AppRouteWhitelist
	case AppRouteBlacklist, "deny", "except":
		return AppRouteBlacklist
	default:
		return AppRouteOff
	}
}

// ParseAppList extracts unique process names / paths from a multiline list.
func ParseAppList(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name := NormalizeProcessName(line)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

// NormalizeProcessName turns a path or label into a PROCESS-NAME / PROCESS-PATH value.
func NormalizeProcessName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	if s == "" {
		return ""
	}
	// Full path → keep as PROCESS-PATH candidate (contains separators).
	if strings.ContainsAny(s, `/\`) {
		return filepath.Clean(s)
	}
	base := filepath.Base(s)
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

// AppendAppProcessRules adds PROCESS-NAME / PROCESS-PATH rules for per-app split tunneling.
func AppendAppProcessRules(rules []string, in BuildInput) []string {
	mode := NormalizeAppRouteMode(in.AppRouteMode)
	if mode == AppRouteOff {
		return rules
	}
	apps := ParseAppList(in.AppRouteList)
	if len(apps) == 0 {
		return rules
	}
	proxyTarget := in.ProxyGroup
	if proxyTarget == "" {
		proxyTarget = defaults.ProxyGroup
	}
	action := proxyTarget
	if mode == AppRouteBlacklist {
		action = "DIRECT"
	}
	for _, app := range apps {
		if strings.ContainsAny(app, `/\`) {
			rules = append(rules, "PROCESS-PATH,"+app+","+action)
		} else {
			rules = append(rules, "PROCESS-NAME,"+app+","+action)
		}
	}
	return rules
}

// AppRoutingEnabled reports whether process-based rules will be emitted.
func AppRoutingEnabled(in BuildInput) bool {
	mode := NormalizeAppRouteMode(in.AppRouteMode)
	return mode != AppRouteOff && len(ParseAppList(in.AppRouteList)) > 0
}
