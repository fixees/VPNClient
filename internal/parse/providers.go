package parse

import (
	"fmt"
	"strings"

	"myinternetvpn/client/internal/profiles"

	"gopkg.in/yaml.v3"
)

// FetchURLFunc downloads a remote proxy-provider body.
type FetchURLFunc func(url string) ([]byte, error)

// ParseClashYAML extracts proxies from a Clash/Clash Meta document (inline only).
func ParseClashYAML(raw []byte) ([]profiles.ProxyNode, error) {
	return ParseClashYAMLWithFetch(raw, nil)
}

// ParseClashYAMLWithFetch also resolves http(s) proxy-providers when fetch is set.
func ParseClashYAMLWithFetch(raw []byte, fetch FetchURLFunc) ([]profiles.ProxyNode, error) {
	raw = stripBOM(raw)
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := extractProxyList(doc["proxies"])
	if fetch != nil {
		extra, err := resolveProxyProviders(doc["proxy-providers"], fetch)
		if err != nil {
			return nil, err
		}
		out = append(out, extra...)
	} else if len(out) == 0 && hasProxyProviders(doc) {
		return nil, fmt.Errorf("clash yaml uses proxy-providers only; refresh via subscription URL to resolve them")
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no proxies found in clash yaml")
	}
	return out, nil
}

func stripBOM(raw []byte) []byte {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return raw[3:]
	}
	return raw
}

func hasProxyProviders(doc map[string]any) bool {
	_, ok := doc["proxy-providers"]
	return ok
}

func extractProxyList(v any) []profiles.ProxyNode {
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	out := make([]profiles.ProxyNode, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		node := profiles.ProxyNode(m)
		name, _ := node["name"].(string)
		typ, _ := node["type"].(string)
		if strings.TrimSpace(name) == "" || strings.TrimSpace(typ) == "" {
			continue
		}
		if norm := normalizeProxyType(typ); norm != "" {
			node["type"] = norm
		}
		out = append(out, node)
	}
	return out
}

func resolveProxyProviders(v any, fetch FetchURLFunc) ([]profiles.ProxyNode, error) {
	providers, ok := v.(map[string]any)
	if !ok || len(providers) == 0 {
		return nil, nil
	}
	var out []profiles.ProxyNode
	var errs []string
	for name, raw := range providers {
		cfg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(fmt.Sprint(cfg["type"])))
		if typ == "" {
			typ = "http"
		}
		switch typ {
		case "http", "https":
			u := strings.TrimSpace(fmt.Sprint(cfg["url"]))
			if u == "" || fetch == nil {
				continue
			}
			body, err := fetch(u)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", name, err))
				continue
			}
			nodes, err := parseProviderBody(body)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", name, err))
				continue
			}
			out = append(out, nodes...)
		case "inline":
			out = append(out, extractProxyList(cfg["payload"])...)
			out = append(out, extractProxyList(cfg["proxies"])...)
		default:
			// file / nested providers: skip (no local path sandbox)
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("proxy-providers failed: %s", strings.Join(errs, "; "))
	}
	return out, nil
}

func parseProviderBody(body []byte) ([]profiles.ProxyNode, error) {
	body = stripBOM(body)
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil, fmt.Errorf("empty provider body")
	}
	// Provider payloads are usually a proxies list YAML, full Clash doc, or share-link list.
	if looksLikeClashYAML(text) || strings.HasPrefix(strings.ToLower(text), "proxies:") {
		nodes, err := ParseClashYAML(body) // no nested provider recursion by default
		if err == nil {
			return nodes, nil
		}
		// Some providers are just a YAML list of proxy maps.
		var list []any
		if err2 := yaml.Unmarshal(body, &list); err2 == nil {
			if nodes := extractProxyList(list); len(nodes) > 0 {
				return nodes, nil
			}
		}
		return nil, err
	}
	if nodes, err := ParseShareLinkList(text); err == nil && len(nodes) > 0 {
		return nodes, nil
	}
	if decoded, err := decodeBase64(text); err == nil {
		decoded = strings.TrimSpace(decoded)
		if nodes, err := ParseShareLinkList(decoded); err == nil && len(nodes) > 0 {
			return nodes, nil
		}
		if looksLikeClashYAML(decoded) {
			return ParseClashYAML([]byte(decoded))
		}
	}
	return nil, fmt.Errorf("unsupported provider format")
}
