package deeplink

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"myinternetvpn/client/internal/defaults"
)

// Action kinds (Incy-compatible verbs).
const (
	KindConnect    = "connect"
	KindDisconnect = "disconnect"
	KindToggle     = "toggle"
	KindOpen       = "open"
	KindImport     = "import"
	KindAdd        = "add"
)

// Action is a parsed myvpn:// (or myinternetvpn://) deep link.
type Action struct {
	Kind string
	Data string // payload for import/add
	Name string // optional profile name (?name=)
	Raw  string
}

// FromArgs finds the first deep-link / share URI in argv.
func FromArgs(args []string) string {
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if IsDeepLink(a) {
			return a
		}
	}
	return ""
}

// IsDeepLink reports whether s uses a registered app scheme.
func IsDeepLink(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, scheme := range Schemes() {
		if strings.HasPrefix(lower, scheme+"://") || strings.HasPrefix(lower, scheme+":") {
			return true
		}
	}
	return false
}

// Schemes returns registered protocol names without ://.
func Schemes() []string {
	return []string{defaults.URLScheme, defaults.URLSchemeAlt}
}

// Parse converts a myvpn://… link into an Action.
func Parse(raw string) (Action, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Action{}, fmt.Errorf("empty deep link")
	}
	lower := strings.ToLower(raw)
	rest := ""
	matched := false
	for _, scheme := range Schemes() {
		prefix := scheme + "://"
		if strings.HasPrefix(lower, prefix) {
			rest = raw[len(prefix):]
			matched = true
			break
		}
		// myvpn:connect (rare)
		colon := scheme + ":"
		if strings.HasPrefix(lower, colon) && !strings.HasPrefix(lower, prefix) {
			rest = raw[len(colon):]
			rest = strings.TrimPrefix(rest, "//")
			matched = true
			break
		}
	}
	if !matched {
		return Action{}, fmt.Errorf("not a %s deep link", defaults.URLScheme)
	}

	rest = strings.TrimSpace(rest)
	name := ""
	// Optional query on the outer link: ?name=Foo (only when no nested :// after ?)
	if i := strings.Index(rest, "?"); i >= 0 {
		pathPart := rest[:i]
		qPart := rest[i+1:]
		// If path already embeds http(s)/vless/… keep query attached to payload.
		if !strings.Contains(pathPart, "://") {
			rest = pathPart
			if vals, err := url.ParseQuery(qPart); err == nil {
				name = strings.TrimSpace(vals.Get("name"))
				if d := strings.TrimSpace(vals.Get("data")); d != "" && rest == "" {
					rest = "import/" + d
				}
				if d := strings.TrimSpace(vals.Get("url")); d != "" {
					if rest == "" || rest == "import" || rest == "add" {
						rest = strings.TrimSuffix(rest, "/") + "/" + d
						if !strings.Contains(rest, "/") {
							rest = "import/" + d
						}
					}
				}
			}
		}
	}

	rest = strings.Trim(rest, "/")
	if rest == "" {
		return Action{Kind: KindOpen, Name: name, Raw: raw}, nil
	}

	head, payload, hasPayload := strings.Cut(rest, "/")
	headLower := strings.ToLower(head)
	act := Action{Name: name, Raw: raw, Data: strings.TrimSpace(payload)}

	switch headLower {
	case "connect":
		act.Kind = KindConnect
		return act, nil
	case "disconnect", "close":
		act.Kind = KindDisconnect
		return act, nil
	case "toggle":
		act.Kind = KindToggle
		return act, nil
	case "open", "status":
		act.Kind = KindOpen
		return act, nil
	case "import":
		act.Kind = KindImport
		if !hasPayload || act.Data == "" {
			return Action{}, fmt.Errorf("import requires data")
		}
		act.Data = decodePayload(act.Data)
		return act, nil
	case "add":
		act.Kind = KindAdd
		if !hasPayload || act.Data == "" {
			return Action{}, fmt.Errorf("add requires data")
		}
		act.Data = decodePayload(act.Data)
		return act, nil
	default:
		// Unknown verb → treat entire rest as import payload (subscription / config).
		act.Kind = KindImport
		act.Data = decodePayload(rest)
		return act, nil
	}
}

func decodePayload(data string) string {
	data = strings.TrimSpace(data)
	if data == "" {
		return ""
	}
	// If it already looks like a URL / share link / YAML, keep as-is.
	lower := strings.ToLower(data)
	if strings.Contains(lower, "://") ||
		strings.HasPrefix(lower, "proxies:") ||
		strings.Contains(data, "\n") ||
		strings.HasPrefix(data, "{") ||
		strings.HasPrefix(data, "[") {
		return data
	}
	// Try base64 (Incy-style import/{base64})
	if decoded, err := decodeB64(data); err == nil {
		decoded = strings.TrimSpace(decoded)
		if decoded != "" {
			return decoded
		}
	}
	return data
}

func decodeB64(s string) (string, error) {
	s = strings.TrimSpace(s)
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, enc := range encodings {
		b, err := enc.DecodeString(s)
		if err == nil && len(b) > 0 {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("not base64")
}
