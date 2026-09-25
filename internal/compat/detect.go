package compat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"myinternetvpn/client/internal/winutil"
)

// Catalog is a data-driven list of DPI / packet-filter tools that conflict with TUN split-tunnel.
type Catalog struct {
	Version int    `json:"version"`
	Tools   []Tool `json:"tools"`
}

// Tool describes how to detect a third-party bypass utility and what it "owns".
type Tool struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Match       ToolMatch     `json:"match"`
	Handles     ToolHandles   `json:"handles"`
	WhenOverlap OverlapPolicy `json:"when_overlap"`
}

type ToolMatch struct {
	ProcessNames   []string `json:"process_names"`
	PathSubstrings []string `json:"path_substrings"`
	Drivers        []string `json:"drivers"`
}

type ToolHandles struct {
	AppNames       []string `json:"app_names"`
	DomainKeywords []string `json:"domain_keywords"`
}

type OverlapPolicy struct {
	// warn | warn_and_prefer_direct
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Finding is one detected tool + optional overlap with the user's route lists.
type Finding struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Severity       string   `json:"severity"`
	Message        string   `json:"message"`
	MatchedBy      []string `json:"matchedBy"`
	OverlapApps    []string `json:"overlapApps,omitempty"`
	OverlapDomains []string `json:"overlapDomains,omitempty"`
	PreferDirect   bool     `json:"preferDirect"`
	RunningProcess string   `json:"runningProcess,omitempty"`
}

// RouteContext is the user's current split-tunnel selection.
type RouteContext struct {
	Mode    string   // off | whitelist | blacklist
	Apps    []string // PROCESS-NAME entries
	Domains []string // domain rules from RouteProxy (force-proxy list)
}

var (
	catalogMu   sync.RWMutex
	catalogData Catalog
	catalogOK   bool
)

// LoadCatalog parses JSON signatures from disk.
func LoadCatalog(path string) (Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	return ParseCatalog(raw)
}

// ParseCatalog decodes a conflicts catalog.
func ParseCatalog(raw []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// SetDefaultCatalog installs the signature pack used by DetectDefault.
func SetDefaultCatalog(c Catalog) {
	catalogMu.Lock()
	catalogData = c
	catalogOK = true
	catalogMu.Unlock()
}

// SetDefaultCatalogJSON parses and installs embedded JSON.
func SetDefaultCatalogJSON(raw []byte) error {
	c, err := ParseCatalog(raw)
	if err != nil {
		return err
	}
	SetDefaultCatalog(c)
	return nil
}

// DefaultCatalog returns the installed signature pack.
func DefaultCatalog() (Catalog, bool) {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	return catalogData, catalogOK
}

// Detect scans running processes against the catalog.
func Detect(cat Catalog, apps []winutil.RunningApp, route RouteContext) []Finding {
	route.Mode = strings.ToLower(strings.TrimSpace(route.Mode))
	routeSet := map[string]string{}
	for _, a := range route.Apps {
		base := filepath.Base(strings.TrimSpace(a))
		if base == "" {
			continue
		}
		routeSet[strings.ToLower(base)] = base
	}

	var out []Finding
	for _, tool := range cat.Tools {
		hit, how, proc := matchTool(tool, apps)
		if !hit {
			continue
		}
		f := Finding{
			ID:             tool.ID,
			Title:          tool.Title,
			Severity:       tool.WhenOverlap.Severity,
			Message:        tool.WhenOverlap.Message,
			MatchedBy:      how,
			RunningProcess: proc,
			PreferDirect:   strings.EqualFold(tool.WhenOverlap.Severity, "warn_and_prefer_direct"),
		}
		if f.Severity == "" {
			f.Severity = "warn"
		}
		if f.Title == "" {
			f.Title = tool.ID
		}
		if route.Mode == "whitelist" && len(tool.Handles.AppNames) > 0 {
			for _, name := range tool.Handles.AppNames {
				key := strings.ToLower(filepath.Base(name))
				if display, ok := routeSet[key]; ok {
					f.OverlapApps = append(f.OverlapApps, display)
				}
			}
		}
		for _, kw := range tool.Handles.DomainKeywords {
			kw = strings.ToLower(strings.TrimSpace(kw))
			if kw == "" {
				continue
			}
			for _, dom := range route.Domains {
				line := strings.ToLower(strings.TrimSpace(dom))
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if strings.Contains(line, kw) {
					f.OverlapDomains = append(f.OverlapDomains, strings.TrimSpace(dom))
				}
			}
		}
		f.OverlapDomains = uniq(f.OverlapDomains)
		out = append(out, f)
	}
	return out
}

// DetectDefault uses the installed catalog.
func DetectDefault(apps []winutil.RunningApp, route RouteContext) []Finding {
	cat, ok := DefaultCatalog()
	if !ok {
		return nil
	}
	return Detect(cat, apps, route)
}

func matchTool(tool Tool, apps []winutil.RunningApp) (bool, []string, string) {
	var how []string
	procName := ""
	nameSet := map[string]struct{}{}
	for _, n := range tool.Match.ProcessNames {
		nameSet[strings.ToLower(filepath.Base(n))] = struct{}{}
	}
	for _, a := range apps {
		base := strings.ToLower(filepath.Base(a.Name))
		path := strings.ToLower(a.Path)
		if _, ok := nameSet[base]; ok {
			how = append(how, "process:"+a.Name)
			if procName == "" {
				procName = a.Name
			}
		}
		for _, sub := range tool.Match.PathSubstrings {
			sub = strings.ToLower(strings.TrimSpace(sub))
			if sub != "" && path != "" && strings.Contains(path, sub) {
				how = append(how, "path:"+sub)
				if procName == "" {
					procName = a.Name
				}
			}
		}
	}
	for _, drv := range tool.Match.Drivers {
		if winutil.DriverServiceRunning(drv) {
			how = append(how, "driver:"+drv)
		}
	}
	// Driver alone is too noisy (WinDivert is shared). Need process or path.
	strong := false
	for _, h := range how {
		if strings.HasPrefix(h, "process:") || strings.HasPrefix(h, "path:") {
			strong = true
			break
		}
	}
	if !strong {
		return false, nil, ""
	}
	return true, uniq(how), procName
}

func uniq(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// SuggestDirectApps returns overlapped app names that should stay off the VPN whitelist.
func SuggestDirectApps(findings []Finding) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, f := range findings {
		if !f.PreferDirect {
			continue
		}
		for _, a := range f.OverlapApps {
			key := strings.ToLower(a)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, a)
		}
	}
	return out
}

// SuggestDirectDomainLines returns RouteProxy lines that should not force PROXY
// while a PreferDirect DPI tool is active.
func SuggestDirectDomainLines(findings []Finding) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, f := range findings {
		if !f.PreferDirect {
			continue
		}
		for _, d := range f.OverlapDomains {
			key := strings.ToLower(strings.TrimSpace(d))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, d)
		}
	}
	return out
}

// ExclusionKey fingerprints the set of apps/domains kept DIRECT by compat.
func ExclusionKey(findings []Finding) string {
	apps := SuggestDirectApps(findings)
	doms := SuggestDirectDomainLines(findings)
	parts := make([]string, 0, len(apps)+len(doms)+1)
	for _, f := range findings {
		if f.PreferDirect || len(f.OverlapApps) > 0 || len(f.OverlapDomains) > 0 {
			parts = append(parts, "tool:"+strings.ToLower(f.ID))
		} else {
			parts = append(parts, "tool:"+strings.ToLower(f.ID)+":warn")
		}
	}
	for _, a := range apps {
		parts = append(parts, "app:"+strings.ToLower(a))
	}
	for _, d := range doms {
		parts = append(parts, "dom:"+strings.ToLower(strings.TrimSpace(d)))
	}
	return strings.Join(parts, "|")
}

// RemoveAppsFromList drops names (case-insensitive basename) from a multiline app list.
func RemoveAppsFromList(list string, remove []string) string {
	if len(remove) == 0 {
		return list
	}
	drop := map[string]struct{}{}
	for _, r := range remove {
		drop[strings.ToLower(filepath.Base(strings.TrimSpace(r)))] = struct{}{}
	}
	var keep []string
	for _, line := range strings.Split(list, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			keep = append(keep, line)
			continue
		}
		if strings.HasPrefix(trim, "#") {
			keep = append(keep, line)
			continue
		}
		base := strings.ToLower(filepath.Base(trim))
		if _, ok := drop[base]; ok {
			continue
		}
		keep = append(keep, line)
	}
	return strings.TrimRight(strings.Join(keep, "\n"), "\n")
}

// RemoveLinesFromList drops exact (case-insensitive trim) lines from a multiline list.
func RemoveLinesFromList(list string, remove []string) string {
	if len(remove) == 0 {
		return list
	}
	drop := map[string]struct{}{}
	for _, r := range remove {
		drop[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	var keep []string
	for _, line := range strings.Split(list, "\n") {
		trim := strings.TrimSpace(line)
		if trim != "" && !strings.HasPrefix(trim, "#") {
			if _, ok := drop[strings.ToLower(trim)]; ok {
				continue
			}
		}
		keep = append(keep, line)
	}
	return strings.TrimRight(strings.Join(keep, "\n"), "\n")
}
