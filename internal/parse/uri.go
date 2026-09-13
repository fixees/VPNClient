package parse

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"myinternetvpn/client/internal/profiles"

	"gopkg.in/yaml.v3"
)

// ParseShareLink converts vless:// vmess:// ss:// into a Clash proxy map.
func ParseShareLink(raw string) (profiles.ProxyNode, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "ss://"):
		return parseShadowsocks(raw)
	case strings.HasPrefix(raw, "vmess://"):
		return parseVMess(raw)
	case strings.HasPrefix(raw, "vless://"):
		return parseVLESS(raw)
	default:
		return nil, fmt.Errorf("unsupported share link scheme")
	}
}

// ParseClashYAML extracts proxies from a Clash/Clash Meta document.
func ParseClashYAML(raw []byte) ([]profiles.ProxyNode, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	list, ok := doc["proxies"].([]any)
	if !ok || len(list) == 0 {
		return nil, fmt.Errorf("no proxies found in clash yaml")
	}
	out := make([]profiles.ProxyNode, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("proxy #%d is not a map", i)
		}
		node := profiles.ProxyNode(m)
		if name, _ := node["name"].(string); name == "" {
			return nil, fmt.Errorf("proxy #%d missing name", i)
		}
		if typ, _ := node["type"].(string); typ == "" {
			return nil, fmt.Errorf("proxy %q missing type", node["name"])
		}
		out = append(out, node)
	}
	return out, nil
}

// ParseInput auto-detects JSON proxy, share link, or Clash YAML.
func ParseInput(raw string) ([]profiles.ProxyNode, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty input")
	}
	if strings.HasPrefix(raw, "{") {
		var node profiles.ProxyNode
		if err := json.Unmarshal([]byte(raw), &node); err != nil {
			return nil, err
		}
		return []profiles.ProxyNode{node}, nil
	}
	if strings.HasPrefix(raw, "[") {
		var nodes []profiles.ProxyNode
		if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
			return nil, err
		}
		return nodes, nil
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "proxies:") || strings.Contains(lower, "\nproxies:") || strings.HasPrefix(lower, "mixed-port:") || strings.HasPrefix(lower, "port:") {
		return ParseClashYAML([]byte(raw))
	}
	// Multi-line share links
	if strings.Count(raw, "\n") > 0 {
		return ParseShareLinkList(raw)
	}
	if strings.Contains(raw, "://") {
		node, err := ParseShareLink(raw)
		if err != nil {
			return nil, err
		}
		return []profiles.ProxyNode{node}, nil
	}
	return ParseClashYAML([]byte(raw))
}

// ParseSubscriptionBody accepts Clash YAML, base64 link lists, or plain share links.
func ParseSubscriptionBody(raw []byte) ([]profiles.ProxyNode, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return nil, fmt.Errorf("empty subscription body")
	}

	// Clash / Meta YAML
	if looksLikeClashYAML(text) {
		if nodes, err := ParseClashYAML([]byte(text)); err == nil {
			return nodes, nil
		}
	}

	// Whole-body base64 (common for v2ray subscriptions)
	if decoded, err := decodeBase64(text); err == nil {
		decoded = strings.TrimSpace(decoded)
		if looksLikeClashYAML(decoded) {
			if nodes, err := ParseClashYAML([]byte(decoded)); err == nil {
				return nodes, nil
			}
		}
		if nodes, err := ParseShareLinkList(decoded); err == nil && len(nodes) > 0 {
			return nodes, nil
		}
	}

	if nodes, err := ParseShareLinkList(text); err == nil && len(nodes) > 0 {
		return nodes, nil
	}
	return nil, fmt.Errorf("unsupported subscription format")
}

// ParseShareLinkList parses one share link per line.
func ParseShareLinkList(raw string) ([]profiles.ProxyNode, error) {
	lines := strings.Split(raw, "\n")
	out := make([]profiles.ProxyNode, 0, len(lines))
	var firstErr error
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		node, err := ParseShareLink(line)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		out = append(out, node)
	}
	if len(out) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("no share links found")
	}
	return out, nil
}

func looksLikeClashYAML(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "proxies:") ||
		strings.HasPrefix(lower, "mixed-port:") ||
		strings.HasPrefix(lower, "port:") ||
		strings.Contains(lower, "proxy-groups:")
}

func parseShadowsocks(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := u.Fragment
	if name == "" {
		name = "ss-node"
	}
	var userinfo string
	host := u.Host
	if u.User != nil {
		pass, hasPass := u.User.Password()
		userinfo = u.User.Username()
		if hasPass {
			userinfo += ":" + pass
		} else if decoded, err := decodeBase64(userinfo); err == nil && strings.Contains(decoded, ":") {
			userinfo = decoded
		}
	} else {
		// ss://BASE64@host:port or ss://BASE64#name
		rest := strings.TrimPrefix(raw, "ss://")
		rest = strings.SplitN(rest, "#", 2)[0]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			decoded, err := decodeBase64(rest[:at])
			if err != nil {
				return nil, err
			}
			userinfo = decoded
			host = rest[at+1:]
		} else {
			decoded, err := decodeBase64(rest)
			if err != nil {
				return nil, err
			}
			// method:password@host:port
			parts := strings.SplitN(decoded, "@", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid ss link")
			}
			userinfo = parts[0]
			host = parts[1]
		}
	}
	method, password, ok := strings.Cut(userinfo, ":")
	if !ok {
		return nil, fmt.Errorf("invalid ss userinfo")
	}
	h, portStr, err := splitHostPort(host)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(portStr)
	return profiles.ProxyNode{
		"name":     name,
		"type":     "ss",
		"server":   h,
		"port":     port,
		"cipher":   method,
		"password": password,
	}, nil
}

func parseVMess(raw string) (profiles.ProxyNode, error) {
	payload := strings.TrimPrefix(raw, "vmess://")
	decoded, err := decodeBase64(payload)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(decoded), &m); err != nil {
		return nil, err
	}
	name, _ := m["ps"].(string)
	if name == "" {
		name = "vmess-node"
	}
	port := toInt(m["port"])
	aid := toInt(m["aid"])
	node := profiles.ProxyNode{
		"name":       name,
		"type":       "vmess",
		"server":     fmt.Sprint(m["add"]),
		"port":       port,
		"uuid":       fmt.Sprint(m["id"]),
		"alterId":    aid,
		"cipher":     "auto",
		"tls":        ternary(fmt.Sprint(m["tls"]) == "tls", true, false),
		"network":    fmt.Sprint(m["net"]),
		"servername": fmt.Sprint(m["sni"]),
	}
	if host, _ := m["host"].(string); host != "" {
		node["ws-opts"] = map[string]any{"path": fmt.Sprint(m["path"]), "headers": map[string]any{"Host": host}}
	} else if path, _ := m["path"].(string); path != "" {
		node["ws-opts"] = map[string]any{"path": path}
	}
	return node, nil
}

func parseVLESS(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := u.Fragment
	if name == "" {
		name = "vless-node"
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	uuid := ""
	if u.User != nil {
		uuid = u.User.Username()
	}
	q := u.Query()
	node := profiles.ProxyNode{
		"name":   name,
		"type":   "vless",
		"server": host,
		"port":   port,
		"uuid":   uuid,
		"tls":    q.Get("security") == "tls" || q.Get("security") == "reality",
		"network": firstNonEmpty(q.Get("type"), "tcp"),
		"udp":    true,
	}
	if sni := q.Get("sni"); sni != "" {
		node["servername"] = sni
	}
	if fp := q.Get("fp"); fp != "" {
		node["client-fingerprint"] = fp
	}
	if flow := q.Get("flow"); flow != "" {
		node["flow"] = flow
	}
	if q.Get("security") == "reality" {
		node["reality-opts"] = map[string]any{
			"public-key": q.Get("pbk"),
			"short-id":   q.Get("sid"),
		}
	}
	if q.Get("type") == "ws" {
		node["ws-opts"] = map[string]any{
			"path": q.Get("path"),
			"headers": map[string]any{
				"Host": firstNonEmpty(q.Get("host"), host),
			},
		}
	}
	return node, nil
}

func decodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		b, err := enc.DecodeString(s)
		if err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("invalid base64")
}

func splitHostPort(hostport string) (string, string, error) {
	if strings.HasPrefix(hostport, "[") {
		return netSplitHostPort(hostport)
	}
	h, p, ok := strings.Cut(hostport, ":")
	if !ok {
		return "", "", fmt.Errorf("missing port")
	}
	return h, p, nil
}

func netSplitHostPort(hostport string) (string, string, error) {
	u, err := url.Parse("scheme://" + hostport)
	if err != nil {
		return "", "", err
	}
	return u.Hostname(), u.Port(), nil
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		n, _ := strconv.Atoi(fmt.Sprint(v))
		return n
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
