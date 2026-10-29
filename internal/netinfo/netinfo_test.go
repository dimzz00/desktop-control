// SPDX-License-Identifier: AGPL-3.0-or-later

package netinfo

import (
	"net"
	"strings"
	"testing"
)

// ipv4Net builds a *net.IPNet for an IPv4 address, the concrete type that
// net.Interface.Addrs returns on Linux.
func ipv4Net(t *testing.T, s string) *net.IPNet {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("bad test IP %q", s)
	}
	if ip.To4() == nil {
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
}

// wantAddress renders an Address for readable diffs.
func wantAddress(a Address) string {
	return string(a.Kind) + " " + a.IP.String() + " " + a.Iface
}

func gotAddresses(got []Address) []string {
	out := make([]string, len(got))
	for i, a := range got {
		out[i] = wantAddress(a)
	}
	return out
}

func assertAddresses(t *testing.T, got []Address, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d addresses %v, want %d %v", len(got), gotAddresses(got), len(want), want)
	}
	for i := range want {
		if gotAddresses(got)[i] != want[i] {
			t.Fatalf("address %d = %q, want %q (all: %v)", i, gotAddresses(got)[i], want[i], gotAddresses(got))
		}
	}
}

// TestClassify covers the brief's table: a wireless LAN address, an Android USB
// tethering address, a tailnet address, and two interface kinds that must be
// ignored (docker0 and loopback).
func TestClassify(t *testing.T) {
	ifaces := []net.Interface{
		{Name: "wlp2s0", Flags: net.FlagUp},
		{Name: "enx00e04c", Flags: net.FlagUp},
		{Name: "tailscale0", Flags: net.FlagUp},
		{Name: "docker0", Flags: net.FlagUp},
		{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
	}
	addrs := map[string][]net.Addr{
		"wlp2s0":     {ipv4Net(t, "192.168.1.42")},
		"enx00e04c":  {ipv4Net(t, "192.168.42.129")},
		"tailscale0": {ipv4Net(t, "100.101.102.103")},
		"docker0":    {ipv4Net(t, "172.17.0.1")},
		"lo":         {ipv4Net(t, "127.0.0.1")},
	}

	got := Classify(ifaces, addrs)
	assertAddresses(t, got, []string{
		"lan 192.168.1.42 wlp2s0",
		"usb 192.168.42.129 enx00e04c",
		"tailnet 100.101.102.103 tailscale0",
	})
}

// TestClassifyVirtualInterfacesSkipped verifies the virtual-interface prefix
// list and the down/loopback flags. None of these may ever reach `rc status`.
func TestClassifyVirtualInterfacesSkipped(t *testing.T) {
	names := []string{
		"docker0", "docker_gwbridge", "virbr0", "virbr0-nic", "veth1234",
		"br-abcdef", "vmnet1", "vmnet8", "lo",
	}
	ifaces := make([]net.Interface, 0, len(names)+2)
	addrs := make(map[string][]net.Addr, len(names)+2)
	for _, name := range names {
		ifaces = append(ifaces, net.Interface{Name: name, Flags: net.FlagUp})
		addrs[name] = []net.Addr{ipv4Net(t, "10.9.9.9")}
	}
	ifaces = append(ifaces,
		net.Interface{Name: "eth0", Flags: 0},
		net.Interface{Name: "eth1", Flags: net.FlagUp | net.FlagLoopback},
	)
	addrs["eth0"] = []net.Addr{ipv4Net(t, "10.0.0.2")}
	addrs["eth1"] = []net.Addr{ipv4Net(t, "10.0.0.3")}

	assertAddresses(t, Classify(ifaces, addrs), nil)
}

// TestClassifyTailnetByRange pins rule 4: the 100.64.0.0/10 range alone decides
// tailnet, whatever the interface is called.
func TestClassifyTailnetByRange(t *testing.T) {
	ifaces := []net.Interface{{Name: "enp5s0", Flags: net.FlagUp}}
	addrs := map[string][]net.Addr{"enp5s0": {ipv4Net(t, "100.64.0.5")}}

	assertAddresses(t, Classify(ifaces, addrs), []string{"tailnet 100.64.0.5 enp5s0"})
}

// TestClassifyUSBByInterfaceName pins the second half of rule 4: a USB-style
// interface name forces usb even for an address that is otherwise RFC1918 LAN.
func TestClassifyUSBByInterfaceName(t *testing.T) {
	tests := []string{"usb0", "rndis0", "enx001122334455"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			ifaces := []net.Interface{{Name: name, Flags: net.FlagUp}}
			addrs := map[string][]net.Addr{name: {ipv4Net(t, "10.23.0.1")}}

			assertAddresses(t, Classify(ifaces, addrs), []string{"usb 10.23.0.1 " + name})
		})
	}
}

// TestClassifySkipsUnusableAddresses covers rule 3: IPv6, link-local IPv4, and
// the unspecified address are all dropped.
func TestClassifySkipsUnusableAddresses(t *testing.T) {
	ifaces := []net.Interface{{Name: "eth0", Flags: net.FlagUp}}
	addrs := map[string][]net.Addr{"eth0": {
		ipv4Net(t, "169.254.10.20"),
		ipv4Net(t, "0.0.0.0"),
		ipv4Net(t, "fe80::1"),
		ipv4Net(t, "2001:db8::1"),
	}}

	got := Classify(ifaces, addrs)
	if got == nil {
		t.Fatal("Classify returned nil, want non-nil empty slice")
	}
	assertAddresses(t, got, nil)
}

// TestClassifyMultipleAddressesOneInterface covers rule 6: every IPv4 address of
// one interface is reported, sorted by address bytes.
func TestClassifyMultipleAddressesOneInterface(t *testing.T) {
	ifaces := []net.Interface{{Name: "wlp2s0", Flags: net.FlagUp}}
	addrs := map[string][]net.Addr{"wlp2s0": {
		ipv4Net(t, "192.168.1.44"),
		ipv4Net(t, "10.0.0.5"),
		ipv4Net(t, "192.168.1.43"),
	}}

	assertAddresses(t, Classify(ifaces, addrs), []string{
		"lan 10.0.0.5 wlp2s0",
		"lan 192.168.1.43 wlp2s0",
		"lan 192.168.1.44 wlp2s0",
	})
}

// TestClassifySortOrder covers rule 5 end to end: kind order lan, usb, tailnet,
// other, then interface name ascending, then IP bytes ascending.
func TestClassifySortOrder(t *testing.T) {
	ifaces := []net.Interface{
		{Name: "wg0", Flags: net.FlagUp},
		{Name: "tailscale0", Flags: net.FlagUp},
		{Name: "usb0", Flags: net.FlagUp},
		{Name: "eth9", Flags: net.FlagUp},
		{Name: "eth0", Flags: net.FlagUp},
		{Name: "docker0", Flags: net.FlagUp},
	}
	addrs := map[string][]net.Addr{
		"wg0":        {ipv4Net(t, "203.0.113.5")},
		"tailscale0": {ipv4Net(t, "100.100.0.1")},
		"usb0":       {ipv4Net(t, "10.0.0.2")},
		"eth9":       {ipv4Net(t, "10.1.1.1")},
		"eth0":       {ipv4Net(t, "192.168.1.2")},
		"docker0":    {ipv4Net(t, "172.17.0.1")},
	}

	assertAddresses(t, Classify(ifaces, addrs), []string{
		"lan 192.168.1.2 eth0",
		"lan 10.1.1.1 eth9",
		"usb 10.0.0.2 usb0",
		"tailnet 100.100.0.1 tailscale0",
		"other 203.0.113.5 wg0",
	})
}

// TestClassifyAddrTypes accepts the two net.Addr shapes that carry an IP and
// ignores implementations it cannot interpret.
func TestClassifyAddrTypes(t *testing.T) {
	ifaces := []net.Interface{{Name: "eth0", Flags: net.FlagUp}}
	addrs := map[string][]net.Addr{"eth0": {
		&net.IPAddr{IP: net.ParseIP("192.168.7.8")},
		unknownAddr("garbage"),
	}}

	assertAddresses(t, Classify(ifaces, addrs), []string{"lan 192.168.7.8 eth0"})
}

// unknownAddr is a net.Addr that carries no parsable IP.
type unknownAddr string

func (a unknownAddr) Network() string { return "test" }
func (a unknownAddr) String() string  { return string(a) }

// TestClassifyDeduplicatesAddresses covers rule 8: an address listed twice on
// the same interface appears once.
func TestClassifyDeduplicatesAddresses(t *testing.T) {
	ifaces := []net.Interface{{Name: "wlp2s0", Flags: net.FlagUp}}
	addrs := map[string][]net.Addr{"wlp2s0": {
		ipv4Net(t, "192.168.1.42"),
		ipv4Net(t, "192.168.1.42"),
	}}

	assertAddresses(t, Classify(ifaces, addrs), []string{"lan 192.168.1.42 wlp2s0"})
}

// TestDetect checks the real machine. It must never surface loopback or a
// virtual interface, and every address must be usable IPv4.
func TestDetect(t *testing.T) {
	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil, want non-nil slice")
	}

	for _, a := range got {
		if a.IP.To4() == nil {
			t.Errorf("Detect returned non-IPv4 address %v", a.IP)
		}
		if a.IP.IsLoopback() {
			t.Errorf("Detect returned loopback address %v on %q", a.IP, a.Iface)
		}
		if a.IP.IsLinkLocalUnicast() || a.IP.IsUnspecified() {
			t.Errorf("Detect returned unusable address %v on %q", a.IP, a.Iface)
		}
		if isVirtualInterfaceName(a.Iface) {
			t.Errorf("Detect returned virtual interface %q", a.Iface)
		}
		if strings.HasPrefix(a.Iface, "lo") {
			t.Errorf("Detect returned loopback interface %q", a.Iface)
		}
		switch a.Kind {
		case KindLAN, KindUSB, KindTailnet, KindOther:
		default:
			t.Errorf("Detect returned unknown kind %q", a.Kind)
		}
	}
}
