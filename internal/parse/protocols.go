package parse

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"myinternetvpn/client/internal/profiles"
)

// parseTrojan converts trojan://password@host:port?...#name
func parseTrojan(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "trojan-node"
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
		if p, ok := u.User.Password(); ok {
			password = u.User.Username() + ":" + p
		}
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	node := profiles.ProxyNode{
		"name":     name,
		"type":     "trojan",
		"server":   host,
		"port":     port,
		"password": password,
		"udp":      true,
	}
	if sni := firstNonEmpty(q.Get("sni"), q.Get("peer")); sni != "" {
		node["sni"] = sni
	}
	if alpn := q.Get("alpn"); alpn != "" {
		node["alpn"] = splitCSV(alpn)
	}
	if fp := q.Get("fp"); fp != "" {
		node["client-fingerprint"] = fp
	}
	if skip := truthy(q.Get("allowInsecure"), q.Get("allow_insecure"), q.Get("insecure")); skip {
		node["skip-cert-verify"] = true
	}
	network := firstNonEmpty(q.Get("type"), q.Get("network"))
	if network != "" && network != "tcp" {
		node["network"] = network
	}
	applyTransportOpts(node, q, host)
	if q.Get("security") == "reality" || q.Get("pbk") != "" {
		node["reality-opts"] = map[string]any{
			"public-key": q.Get("pbk"),
			"short-id":   q.Get("sid"),
		}
	}
	return node, nil
}

// parseTUIC converts tuic://uuid:password@host:port?...#name
func parseTUIC(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "tuic-node"
	}
	uuid, password := "", ""
	if u.User != nil {
		uuid = u.User.Username()
		password, _ = u.User.Password()
		if password == "" {
			password = firstNonEmpty(u.Query().Get("password"), u.Query().Get("pass"))
		}
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	node := profiles.ProxyNode{
		"name":     name,
		"type":     "tuic",
		"server":   host,
		"port":     port,
		"uuid":     uuid,
		"password": password,
		"udp":      true,
	}
	if sni := firstNonEmpty(q.Get("sni"), q.Get("peer")); sni != "" {
		node["sni"] = sni
	}
	if alpn := q.Get("alpn"); alpn != "" {
		node["alpn"] = splitCSV(alpn)
	}
	if cc := firstNonEmpty(q.Get("congestion_control"), q.Get("congestion-controller"), q.Get("cc")); cc != "" {
		node["congestion-controller"] = cc
	}
	if mode := firstNonEmpty(q.Get("udp_relay_mode"), q.Get("udp-relay-mode")); mode != "" {
		node["udp-relay-mode"] = mode
	}
	if skip := truthy(q.Get("allow_insecure"), q.Get("allowInsecure"), q.Get("insecure")); skip {
		node["skip-cert-verify"] = true
	}
	if v := q.Get("reduce_rtt"); v != "" {
		node["reduce-rtt"] = truthy(v)
	}
	return node, nil
}

// parseHysteria2 converts hysteria2:// / hy2:// password@host:port?...#name
func parseHysteria2(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "hysteria2-node"
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
		if p, ok := u.User.Password(); ok && p != "" {
			password = u.User.Username() + ":" + p
		}
	}
	if password == "" {
		password = firstNonEmpty(u.Query().Get("auth"), u.Query().Get("password"))
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	node := profiles.ProxyNode{
		"name":     name,
		"type":     "hysteria2",
		"server":   host,
		"port":     port,
		"password": password,
		"udp":      true,
	}
	if sni := firstNonEmpty(q.Get("sni"), q.Get("peer")); sni != "" {
		node["sni"] = sni
	}
	if obfs := q.Get("obfs"); obfs != "" {
		node["obfs"] = obfs
	}
	if op := firstNonEmpty(q.Get("obfs-password"), q.Get("obfs_password")); op != "" {
		node["obfs-password"] = op
	}
	if skip := truthy(q.Get("insecure"), q.Get("allowInsecure"), q.Get("allow_insecure")); skip {
		node["skip-cert-verify"] = true
	}
	if pin := firstNonEmpty(q.Get("pinSHA256"), q.Get("pin-sha256")); pin != "" {
		node["fingerprint"] = pin
	}
	return node, nil
}

// parseHysteria converts hysteria://host:port?auth=... (v1)
func parseHysteria(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "hysteria-node"
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	auth := firstNonEmpty(q.Get("auth"), q.Get("auth_str"), q.Get("peerAuth"))
	if u.User != nil && auth == "" {
		auth = u.User.Username()
	}
	node := profiles.ProxyNode{
		"name":   name,
		"type":   "hysteria",
		"server": host,
		"port":   port,
		"udp":    true,
	}
	if auth != "" {
		node["auth-str"] = auth
	}
	if sni := firstNonEmpty(q.Get("peer"), q.Get("sni")); sni != "" {
		node["sni"] = sni
	}
	if up := firstNonEmpty(q.Get("upmbps"), q.Get("up")); up != "" {
		node["up"] = up
	}
	if down := firstNonEmpty(q.Get("downmbps"), q.Get("down")); down != "" {
		node["down"] = down
	}
	if proto := q.Get("protocol"); proto != "" {
		node["protocol"] = proto
	}
	if skip := truthy(q.Get("insecure"), q.Get("allowInsecure")); skip {
		node["skip-cert-verify"] = true
	}
	if alpn := q.Get("alpn"); alpn != "" {
		node["alpn"] = splitCSV(alpn)
	}
	return node, nil
}

// parseWireGuard converts wireguard://privatekey@server:port?publickey=...&address=...
func parseWireGuard(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "wireguard-node"
	}
	privateKey := ""
	if u.User != nil {
		privateKey = u.User.Username()
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	if privateKey == "" {
		privateKey = firstNonEmpty(q.Get("privatekey"), q.Get("private-key"), q.Get("privateKey"))
	}
	pub := firstNonEmpty(q.Get("publickey"), q.Get("public-key"), q.Get("publicKey"))
	addr := firstNonEmpty(q.Get("address"), q.Get("ip"), q.Get("selfip"))
	node := profiles.ProxyNode{
		"name":        name,
		"type":        "wireguard",
		"server":      host,
		"port":        port,
		"private-key": privateKey,
		"public-key":  pub,
		"udp":         true,
		"allowed-ips": []string{"0.0.0.0/0", "::/0"},
	}
	if addr != "" {
		parts := splitCSV(addr)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.Contains(p, ":") {
				node["ipv6"] = p
			} else if p != "" {
				node["ip"] = p
			}
		}
	}
	if mtu := q.Get("mtu"); mtu != "" {
		if n, err := strconv.Atoi(mtu); err == nil {
			node["mtu"] = n
		}
	}
	if psk := firstNonEmpty(q.Get("presharedkey"), q.Get("pre-shared-key"), q.Get("psk")); psk != "" {
		node["pre-shared-key"] = psk
	}
	if reserved := q.Get("reserved"); reserved != "" {
		node["reserved"] = reserved
	}
	return node, nil
}

// parseSSH converts ssh://user:pass@host:port#name
func parseSSH(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "ssh-node"
	}
	user, password := "", ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 22
	}
	q := u.Query()
	node := profiles.ProxyNode{
		"name":     name,
		"type":     "ssh",
		"server":   host,
		"port":     port,
		"username": firstNonEmpty(user, q.Get("user"), q.Get("username")),
	}
	if password != "" {
		node["password"] = password
	}
	if pk := firstNonEmpty(q.Get("private-key"), q.Get("privateKey"), q.Get("key")); pk != "" {
		node["private-key"] = pk
	}
	if pp := firstNonEmpty(q.Get("private-key-passphrase"), q.Get("passphrase")); pp != "" {
		node["private-key-passphrase"] = pp
	}
	if hk := q.Get("host-key"); hk != "" {
		node["host-key"] = splitCSV(hk)
	}
	return node, nil
}

// parseSocks5 converts socks:// / socks5://user:pass@host:port#name
func parseSocks5(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "socks-node"
	}
	user, password := "", ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}
	port, _ := strconv.Atoi(u.Port())
	node := profiles.ProxyNode{
		"name":   name,
		"type":   "socks5",
		"server": u.Hostname(),
		"port":   port,
		"udp":    true,
	}
	if user != "" {
		node["username"] = user
	}
	if password != "" {
		node["password"] = password
	}
	if truthy(u.Query().Get("tls")) {
		node["tls"] = true
	}
	return node, nil
}

// parseHTTPProxy converts http://user:pass@host:port#name as Clash http outbound.
// Only used when fragment/query marks it as a proxy node (not a subscription URL).
func parseHTTPProxy(raw string) (profiles.ProxyNode, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	name := decodeProxyName(u.Fragment)
	if name == "" {
		name = "http-node"
	}
	user, password := "", ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		if u.Scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}
	node := profiles.ProxyNode{
		"name":   name,
		"type":   "http",
		"server": u.Hostname(),
		"port":   port,
	}
	if user != "" {
		node["username"] = user
	}
	if password != "" {
		node["password"] = password
	}
	if u.Scheme == "https" || truthy(u.Query().Get("tls")) {
		node["tls"] = true
	}
	return node, nil
}

func applyTransportOpts(node profiles.ProxyNode, q url.Values, defaultHost string) {
	network := firstNonEmpty(q.Get("type"), q.Get("network"))
	switch network {
	case "ws", "websocket":
		node["network"] = "ws"
		node["ws-opts"] = map[string]any{
			"path": firstNonEmpty(q.Get("path"), "/"),
			"headers": map[string]any{
				"Host": firstNonEmpty(q.Get("host"), q.Get("Host"), defaultHost),
			},
		}
	case "grpc":
		node["network"] = "grpc"
		node["grpc-opts"] = map[string]any{
			"grpc-service-name": firstNonEmpty(q.Get("serviceName"), q.Get("service-name"), q.Get("path")),
		}
	case "h2", "http":
		node["network"] = network
		node["h2-opts"] = map[string]any{
			"host": splitCSV(firstNonEmpty(q.Get("host"), defaultHost)),
			"path": firstNonEmpty(q.Get("path"), "/"),
		}
	}
}

func splitCSV(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truthy(vals ...string) bool {
	for _, v := range vals {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

// normalizeProxyType maps common aliases to mihomo type names.
func normalizeProxyType(typ string) string {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "hy2":
		return "hysteria2"
	case "hy", "hysteria1":
		return "hysteria"
	case "wg":
		return "wireguard"
	case "socks", "socks5":
		return "socks5"
	default:
		return strings.ToLower(strings.TrimSpace(typ))
	}
}

// SupportedShareSchemes lists URI schemes accepted by ParseShareLink.
func SupportedShareSchemes() []string {
	return []string{
		"ss", "vmess", "vless", "trojan",
		"tuic", "hysteria", "hysteria2", "hy2", "hy",
		"wireguard", "wg", "ssh",
		"socks", "socks5",
	}
}

// Ensure unused import doesn't break if fmt needed later for errors.
var _ = fmt.Sprintf
