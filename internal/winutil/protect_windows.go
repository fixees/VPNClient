//go:build windows

package winutil

import (
	"fmt"
	"os/exec"
	"strings"

	"myinternetvpn/client/internal/defaults"
)

const (
	killSwitchRuleBlock = defaults.KillSwitchBlockRule
	killSwitchRuleAllow = defaults.KillSwitchAllowRule
	dnsLeakRule         = defaults.DNSLeakRule
)

// KillSwitch uses Windows Filtering via netsh advfirewall rules.
// This is a practical firewall-based kill switch (not a full custom WFP callout driver).
type KillSwitch struct {
	active bool
}

func NewKillSwitch() *KillSwitch { return &KillSwitch{} }

func (k *KillSwitch) Active() bool { return k.active }

// Enable blocks outbound traffic except the VPN interface (required).
func (k *KillSwitch) Enable(vpnInterface string) error {
	vpnInterface = strings.TrimSpace(vpnInterface)
	if vpnInterface == "" {
		return fmt.Errorf("kill switch requires VPN interface name")
	}
	_ = k.Disable()
	// Allow VPN interface first, then block the rest — avoids a total blackout window.
	if err := runNetsh("advfirewall", "firewall", "add", "rule",
		"name="+killSwitchRuleAllow,
		"dir=out",
		"action=allow",
		"enable=yes",
		"profile=any",
		"interface="+vpnInterface,
	); err != nil {
		return err
	}
	if err := runNetsh("advfirewall", "firewall", "add", "rule",
		"name="+killSwitchRuleBlock,
		"dir=out",
		"action=block",
		"enable=yes",
		"profile=any",
	); err != nil {
		_ = k.Disable()
		return err
	}
	k.active = true
	return nil
}

// Disable removes kill-switch firewall rules.
func (k *KillSwitch) Disable() error {
	_ = runNetsh("advfirewall", "firewall", "delete", "rule", "name="+killSwitchRuleBlock)
	_ = runNetsh("advfirewall", "firewall", "delete", "rule", "name="+killSwitchRuleAllow)
	k.active = false
	return nil
}

// DNSLeakGuard blocks plain DNS (UDP/TCP 53) outbound to force resolver through the tunnel/core.
type DNSLeakGuard struct {
	active bool
}

func NewDNSLeakGuard() *DNSLeakGuard { return &DNSLeakGuard{} }

func (d *DNSLeakGuard) Active() bool { return d.active }

func (d *DNSLeakGuard) Enable() error {
	_ = d.Disable()
	if err := runNetsh("advfirewall", "firewall", "add", "rule",
		"name="+dnsLeakRule,
		"dir=out",
		"action=block",
		"protocol=UDP",
		"remoteport=53",
		"enable=yes",
		"profile=any",
	); err != nil {
		return fmt.Errorf("dns leak udp: %w", err)
	}
	// Also block TCP/53
	if err := runNetsh("advfirewall", "firewall", "add", "rule",
		"name="+dnsLeakRule+"_TCP",
		"dir=out",
		"action=block",
		"protocol=TCP",
		"remoteport=53",
		"enable=yes",
		"profile=any",
	); err != nil {
		_ = d.Disable()
		return fmt.Errorf("dns leak tcp: %w", err)
	}
	d.active = true
	return nil
}

func (d *DNSLeakGuard) Disable() error {
	_ = runNetsh("advfirewall", "firewall", "delete", "rule", "name="+dnsLeakRule)
	_ = runNetsh("advfirewall", "firewall", "delete", "rule", "name="+dnsLeakRule+"_TCP")
	d.active = false
	return nil
}

func runNetsh(args ...string) error {
	cmd := exec.Command("netsh", args...)
	HideConsole(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %s: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
