package config

import (
	"path/filepath"
	"regexp"
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

// AppPathHint is a running (or known) process image used to resolve install dirs.
type AppPathHint struct {
	Name string
	Path string
}

// CollectAppInstallDirs returns unique install roots for listed apps.
// Roots come from full paths in the list and from matching running processes.
// Used to emit PROCESS-PATH-REGEX so Electron helpers (node.exe, *sandbox*.exe, …)
// under the same install tree follow the same VPN/DIRECT policy as the main exe.
func CollectAppInstallDirs(list string, hints []AppPathHint) []string {
	wanted := map[string]struct{}{}
	var explicitDirs []string
	for _, app := range ParseAppList(list) {
		if strings.ContainsAny(app, `/\`) {
			dir := filepath.Clean(filepath.Dir(app))
			if dir != "" && dir != "." {
				explicitDirs = append(explicitDirs, dir)
			}
			wanted[strings.ToLower(filepath.Base(app))] = struct{}{}
			continue
		}
		wanted[strings.ToLower(app)] = struct{}{}
	}
	if len(wanted) == 0 && len(explicitDirs) == 0 {
		return nil
	}

	var dirs []string
	seen := map[string]struct{}{}
	add := func(dir string) {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "" || dir == "." || dir == string(filepath.Separator) {
			return
		}
		// Avoid matching the entire drive.
		if vol := filepath.VolumeName(dir); vol != "" && filepath.Clean(dir) == vol+string(filepath.Separator) {
			return
		}
		key := strings.ToLower(dir)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		dirs = append(dirs, dir)
	}
	for _, d := range explicitDirs {
		add(d)
	}
	for _, h := range hints {
		base := strings.ToLower(filepath.Base(strings.TrimSpace(h.Name)))
		if base == "" {
			base = strings.ToLower(filepath.Base(strings.TrimSpace(h.Path)))
		}
		if _, ok := wanted[base]; !ok {
			continue
		}
		path := strings.TrimSpace(h.Path)
		if path == "" || !strings.ContainsAny(path, `/\`) {
			continue
		}
		add(filepath.Dir(filepath.Clean(path)))
	}
	return dirs
}

// ProcessPathRegex builds a case-insensitive mihomo PROCESS-PATH-REGEX payload
// that matches any executable under installDir (including nested helpers).
func ProcessPathRegex(installDir string) string {
	dir := filepath.Clean(strings.TrimSpace(installDir))
	if dir == "" || dir == "." {
		return ""
	}
	escaped := regexp.QuoteMeta(dir)
	// Accept both Windows and forward slashes after quoting.
	escaped = strings.ReplaceAll(escaped, `\\`, `[\\/]`)
	escaped = strings.ReplaceAll(escaped, `/`, `[\\/]`)
	return "(?i)" + escaped + `[\\/].*`
}

// AppendAppProcessRules adds PROCESS-NAME / PROCESS-PATH / PATH-REGEX rules
// for per-app split tunneling.
func AppendAppProcessRules(rules []string, in BuildInput) []string {
	mode := NormalizeAppRouteMode(in.AppRouteMode)
	if mode == AppRouteOff {
		return rules
	}
	apps := ParseAppList(in.AppRouteList)
	if len(apps) == 0 && len(in.AppInstallDirs) == 0 {
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
	seenRegex := map[string]struct{}{}
	for _, dir := range in.AppInstallDirs {
		rx := ProcessPathRegex(dir)
		if rx == "" {
			continue
		}
		if _, ok := seenRegex[rx]; ok {
			continue
		}
		seenRegex[rx] = struct{}{}
		rules = append(rules, "PROCESS-PATH-REGEX,"+rx+","+action)
	}
	return rules
}

// AppRoutingEnabled reports whether process-based rules will be emitted
// (or whitelist is forced DIRECT after compat emptied the list).
func AppRoutingEnabled(in BuildInput) bool {
	mode := NormalizeAppRouteMode(in.AppRouteMode)
	if mode == AppRouteOff {
		return false
	}
	if AppWhitelistActive(in) {
		return true
	}
	return mode == AppRouteBlacklist && len(ParseAppList(in.AppRouteList)) > 0
}

// AppWhitelistActive is true when whitelist routing applies, including the
// empty-list + KeepDirect case (compat stripped every listed app).
func AppWhitelistActive(in BuildInput) bool {
	if NormalizeAppRouteMode(in.AppRouteMode) != AppRouteWhitelist {
		return false
	}
	if len(ParseAppList(in.AppRouteList)) > 0 {
		return true
	}
	return in.AppWhitelistKeepDirect
}
