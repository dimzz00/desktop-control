// SPDX-License-Identifier: AGPL-3.0-or-later

// Package netinfo classifies the machine's IPv4 addresses so `rc status` can
// tell a phone which address to connect to.
//
// Three paths matter to kendali: the ordinary LAN address of the host, the
// address of an Android USB tethering interface, and the host's tailnet address.
// Everything else is reported as other, and addresses a client could not reach
// (loopback, link-local, and virtual bridge/container interfaces such as
// docker0 or virbr0) are dropped entirely.
package netinfo

import (
	"bytes"
	"net"
	"sort"
	"strings"
)

// Kind labels the network path an address belongs to.
type Kind string

const (
	// KindLAN marks an address on the ordinary local network.
	KindLAN Kind = "lan"
	// KindUSB marks an Android USB tethering address.
	KindUSB Kind = "usb"
	// KindTailnet marks an address on the Tailscale/CGNAT tailnet range.
	KindTailnet Kind = "tailnet"
	// KindOther marks a usable address that is none of the above.
	KindOther Kind = "other"
)

// Address is one classified IPv4 address of the host.
type Address struct {
	// Kind is the network path this address belongs to.
	Kind Kind
	// IP is the IPv4 address. It is always a 4-byte address.
	IP net.IP
	// Iface is the name of the interface that owns the address.
	Iface string
}

// The IPv4 ranges that drive classification.
var (
	tailnetNet = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}
	usbNet     = &net.IPNet{IP: net.IPv4(192, 168, 42, 0), Mask: net.CIDRMask(24, 32)}
	lan10Net   = &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)}
	lan172Net  = &net.IPNet{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)}
	lan192Net  = &net.IPNet{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}
)

// virtualInterfacePrefixes lists the interface name prefixes that belong to
// container runtimes, hypervisors, and virtual bridges. Addresses on these
// interfaces are reachable only from the host itself, so they must never be
// advertised to a client.
var virtualInterfacePrefixes = []string{
	"docker",
	"virbr",
	"veth",
	"br-",
	"vmnet",
}

// usbInterfacePrefixes lists the interface name prefixes used by Android USB
// tethering (gadget, RNDIS, and predictable CD-ROM-style names).
var usbInterfacePrefixes = []string{
	"usb",
	"rndis",
	"enx",
}

// kindOrder gives the sort precedence of each kind: lan, usb, tailnet, other.
func kindOrder(k Kind) int {
	switch k {
	case KindLAN:
		return 0
	case KindUSB:
		return 1
	case KindTailnet:
		return 2
	default:
		return 3
	}
}

// isVirtualInterfaceName reports whether name belongs to a virtual interface
// that should never be advertised.
func isVirtualInterfaceName(name string) bool {
	if name == "lo" {
		return true
	}
	for _, prefix := range virtualInterfacePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// isUSBInterfaceName reports whether name looks like an Android USB tethering
// interface.
func isUSBInterfaceName(name string) bool {
	for _, prefix := range usbInterfacePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// classify assigns one address to a Kind. Range-based rules win over the
// interface name: 100.64.0.0/10 is always tailnet, and a usb-style name always
// wins over the generic RFC1918 rule.
func classify(iface string, ip net.IP) Kind {
	switch {
	case tailnetNet.Contains(ip):
		return KindTailnet
	case usbNet.Contains(ip) || isUSBInterfaceName(iface):
		return KindUSB
	case lan10Net.Contains(ip), lan172Net.Contains(ip), lan192Net.Contains(ip):
		return KindLAN
	default:
		return KindOther
	}
}

// addrIP extracts the IP from the address types net.Interface.Addrs returns. It
// returns nil for any other implementation.
func addrIP(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.IPNet:
		return a.IP
	case *net.IPAddr:
		return a.IP
	default:
		return nil
	}
}

// usableIPv4 reports whether ip is an IPv4 address kendali may advertise: it
// must be IPv4, and not unspecified, loopback, or link-local.
func usableIPv4(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return !v4.IsUnspecified() && !v4.IsLoopback() && !v4.IsLinkLocalUnicast()
}

// Classify turns a snapshot of interfaces and their addresses into the sorted
// list of addresses kendali may advertise. It performs no I/O, so callers can
// test it with synthetic interfaces.
//
// Interfaces that are down, loopback, or virtual (see the docker/virbr/veth/
// br-/vmnet prefixes) are skipped. Only IPv4 addresses are considered, and
// unspecified, loopback, and link-local addresses are dropped. Duplicate
// iface/IP pairs are collapsed. The result is sorted by kind (lan, usb, tailnet,
// other), then interface name, then IP bytes; it is never nil.
func Classify(ifaces []net.Interface, addrs map[string][]net.Addr) []Address {
	out := make([]Address, 0, len(ifaces))
	seen := make(map[string]struct{}, len(ifaces))

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || isVirtualInterfaceName(iface.Name) {
			continue
		}
		for _, addr := range addrs[iface.Name] {
			ip := addrIP(addr)
			if !usableIPv4(ip) {
				continue
			}
			v4 := ip.To4()
			key := iface.Name + "\x00" + string(v4)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, Address{Kind: classify(iface.Name, v4), IP: v4, Iface: iface.Name})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if ka, kb := kindOrder(a.Kind), kindOrder(b.Kind); ka != kb {
			return ka < kb
		}
		if a.Iface != b.Iface {
			return a.Iface < b.Iface
		}
		return bytes.Compare(a.IP, b.IP) < 0
	})

	return out
}

// Detect classifies the addresses of the running machine. It reads the
// interfaces from the operating system and delegates the classification to
// Classify.
//
// On a machine with no usable address the result is an empty, non-nil slice and
// the error is nil. Only failures of net.Interfaces are reported.
func Detect() ([]Address, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	addrs := make(map[string][]net.Addr, len(ifaces))
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || isVirtualInterfaceName(iface.Name) {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		addrs[iface.Name] = ifaceAddrs
	}

	return Classify(ifaces, addrs), nil
}
