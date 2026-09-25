package tests

import (
	"fmt"
	"net"
	"testing"
)

// isPrivateIP checks if IP is in private ranges (RFC1918).
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// Convert to 4-byte representation for easier checking
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	// 10.0.0.0/8
	if ip4[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip4[0] == 192 && ip4[1] == 168 {
		return true
	}
	return false
}

// getLANAddresses mimics the app.go implementation for testing.
func getLANAddresses(port int) []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{}
	}

	var addrs []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range ifaceAddrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			if isPrivateIP(ip) {
				addrs = append(addrs, fmt.Sprintf("%s:%d", ip.String(), port))
			}
		}
	}

	return addrs
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"192.168.255.254", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		{"192.167.1.1", false},
		{"192.169.1.1", false},
		{"127.0.0.1", false},
	}

	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("invalid IP: %s", tc.ip)
		}
		result := isPrivateIP(ip)
		if result != tc.expected {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tc.ip, result, tc.expected)
		}
	}
}

func TestGetLANAddresses(t *testing.T) {
	port := 7890
	addrs := getLANAddresses(port)

	// Test should pass even if no LAN addresses are found (e.g., CI environment)
	// but if addresses are found, they should be valid
	for _, addr := range addrs {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			t.Errorf("invalid address format %q: %v", addr, err)
			continue
		}
		ip := net.ParseIP(host)
		if ip == nil {
			t.Errorf("invalid IP in address %q", addr)
			continue
		}
		if !isPrivateIP(ip) {
			t.Errorf("address %q is not a private IP", addr)
		}
		if portStr != fmt.Sprintf("%d", port) {
			t.Errorf("address %q has wrong port, want %d", addr, port)
		}
	}
}

func TestGetLANAddressesFormat(t *testing.T) {
	// Test that addresses have correct format even with different ports
	for _, port := range []int{7890, 8080, 9999} {
		addrs := getLANAddresses(port)
		for _, addr := range addrs {
			host, portStr, err := net.SplitHostPort(addr)
			if err != nil {
				t.Errorf("port %d: invalid address format %q: %v", port, addr, err)
				continue
			}
			expectedPort := fmt.Sprintf("%d", port)
			if portStr != expectedPort {
				t.Errorf("port %d: address %q has port %s, want %s", port, addr, portStr, expectedPort)
			}
			if net.ParseIP(host) == nil {
				t.Errorf("port %d: invalid IP %q in address", port, host)
			}
		}
	}
}

func TestProxyEndpointFormatting(t *testing.T) {
	// Test PowerShell env format
	port := 7890
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	psScript := fmt.Sprintf("$env:HTTP_PROXY=\"%s\"\n$env:HTTPS_PROXY=\"%s\"\n$env:ALL_PROXY=\"%s\"", proxyURL, proxyURL, proxyURL)

	if !containsAll(psScript, "$env:HTTP_PROXY", "$env:HTTPS_PROXY", "$env:ALL_PROXY") {
		t.Errorf("PowerShell script missing required env vars")
	}

	// Test CMD env format
	cmdScript := fmt.Sprintf("set HTTP_PROXY=%s\nset HTTPS_PROXY=%s\nset ALL_PROXY=%s", proxyURL, proxyURL, proxyURL)
	if !containsAll(cmdScript, "set HTTP_PROXY", "set HTTPS_PROXY", "set ALL_PROXY") {
		t.Errorf("CMD script missing required env vars")
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(sub) == 0 {
			continue
		}
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
