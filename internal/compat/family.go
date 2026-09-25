package compat

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const maxFamilyExes = 8

// ExpandAppFamily returns related .exe basenames from the same install directory
// as exePathOrName. Scans only the binary's own folder (not parent) and caps the
// result to avoid pulling unrelated tools from shared install roots.
func ExpandAppFamily(exePathOrName string) []string {
	raw := strings.TrimSpace(exePathOrName)
	if raw == "" {
		return nil
	}
	base := filepath.Base(raw)
	out := []string{base}

	if !strings.ContainsAny(raw, `/\`) {
		return uniqFold(out)
	}
	dir := filepath.Dir(filepath.Clean(raw))
	if dir == "" || dir == "." || dir == string(filepath.Separator) {
		return uniqFold(out)
	}

	stem := familyStem(base)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return uniqFold(out)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".exe") {
			continue
		}
		lower := strings.ToLower(name)
		if strings.Contains(lower, "uninstall") || strings.Contains(lower, "setup") || strings.Contains(lower, "installer") {
			continue
		}
		// Prefer siblings that share a name stem with the main binary.
		if stem != "" && !strings.Contains(strings.ToLower(familyStem(name)), stem) && !strings.Contains(stem, strings.ToLower(familyStem(name))) {
			// Allow short helpers like Update.exe next to Discord.exe when stem is discord —
			// only keep if stem matches OR name is a common updater colocated (same folder only).
			if !isLikelyHelper(name, stem) {
				continue
			}
		}
		out = append(out, name)
		if len(out) >= maxFamilyExes {
			break
		}
	}
	return uniqFold(out)
}

func familyStem(exe string) string {
	name := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	// Take leading letters only (discord from DiscordPTB / DiscordCanary).
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
			continue
		}
		if b.Len() >= 4 {
			break
		}
		// digit/punct mid-name: stop after we have some letters
		if b.Len() > 0 {
			break
		}
	}
	s := b.String()
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

func isLikelyHelper(exeName, mainStem string) bool {
	lower := strings.ToLower(filepath.Base(exeName))
	stem := familyStem(exeName)
	if mainStem != "" && stem != "" {
		if strings.HasPrefix(stem, mainStem) || strings.HasPrefix(mainStem, stem) {
			return true
		}
		if strings.Contains(stem, mainStem) || strings.Contains(mainStem, stem) {
			return true
		}
	}
	// Co-located updaters / crash handlers without the product name.
	for _, h := range []string{"update", "updater", "crashpad", "helper", "notification"} {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

// MergeAppList appends new basenames into a multiline list without duplicates.
func MergeAppList(list string, add []string) string {
	existing := map[string]struct{}{}
	var lines []string
	for _, line := range strings.Split(list, "\n") {
		trim := strings.TrimSpace(line)
		if trim != "" && !strings.HasPrefix(trim, "#") {
			existing[strings.ToLower(filepath.Base(trim))] = struct{}{}
		}
		if line != "" || len(lines) > 0 {
			lines = append(lines, line)
		}
	}
	for _, a := range add {
		base := filepath.Base(strings.TrimSpace(a))
		if base == "" {
			continue
		}
		key := strings.ToLower(base)
		if _, ok := existing[key]; ok {
			continue
		}
		existing[key] = struct{}{}
		lines = append(lines, base)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func uniqFold(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = filepath.Base(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}
