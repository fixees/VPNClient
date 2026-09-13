//go:build !windows

package winutil

type KillSwitch struct{ active bool }

func NewKillSwitch() *KillSwitch { return &KillSwitch{} }
func (k *KillSwitch) Active() bool { return k.active }
func (k *KillSwitch) Enable(vpnInterface string) error { return ErrNotWindows }
func (k *KillSwitch) Disable() error {
	k.active = false
	return nil
}

type DNSLeakGuard struct{ active bool }

func NewDNSLeakGuard() *DNSLeakGuard { return &DNSLeakGuard{} }
func (d *DNSLeakGuard) Active() bool { return d.active }
func (d *DNSLeakGuard) Enable() error { return ErrNotWindows }
func (d *DNSLeakGuard) Disable() error {
	d.active = false
	return nil
}
