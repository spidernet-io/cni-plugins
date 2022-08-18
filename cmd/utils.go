// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"regexp"
	"strings"
)

var defaultInterfaceName = "eth0"
var defaultMtu = 1500
var defaultConVeth = "veth0"
var sysctlConfPath = "/proc/sys/net/ipv4/conf"

var DefaultInterfacesToExclude = []string{
	"docker.*", "cbr.*", "dummy.*",
	"virbr.*", "lxcbr.*", "veth.*", "lo",
	"cali.*", "tunl.*", "flannel.*", "kube-ipvs.*", "cni.*",
}

// getHostIps return all ip addresses on the node, including ipv4 and ipv6.
// skipping any interfaces whose name matches any of the exclusion list regexes
func getHostIps() ([]string, error) {
	netIfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("[veth] getHostIps: %v ", err)
	}

	var ips []string
	var excludeRegexp *regexp.Regexp

	if excludeRegexp, err = regexp.Compile("(" + strings.Join(DefaultInterfacesToExclude, ")|(") + ")"); err != nil {
		return nil, err
	}

	// Loop through interfaces filtering on the regexes.
	for idx := len(netIfaces) - 1; idx >= 0; idx-- {
		iface := netIfaces[idx]
		exclude := (excludeRegexp != nil) && excludeRegexp.MatchString(iface.Name)
		if !exclude {
			addrs, err := iface.Addrs()
			if err != nil {
				return nil, err
			}
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					continue
				}
				if ip.IsMulticast() || ip.IsLinkLocalUnicast() {
					continue
				}
				ips = append(ips, ip.String())
			}
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("[veth]no one vaild ip on the node")
	}
	return ips, nil
}

// getConIPs return all ip addresses on the eth0 NIC of a given netns, including ipv4 and ipv6
func getConIps(netns ns.NetNS) ([]string, error) {
	ips := make([]string, 0, 2)
	err := netns.Do(func(_ ns.NetNS) error {
		ipv4, ipv6 := false, false
		ifaces, err := net.Interfaces()
		if err != nil {
			return fmt.Errorf("[veth] failed to list interfaces inside pod")
		}
		for _, iface := range ifaces {
			if iface.Name == defaultInterfaceName {
				addrs, err := iface.Addrs()
				if err != nil {
					return err
				}
				for _, addr := range addrs {
					ip, _, err := net.ParseCIDR(addr.String())
					if err != nil {
						return err
					}
					ipv4, ipv6, ips = filterIPs(ip, ipv4, ipv6, ips)
				}
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("[veth] no one vaild ip on the pod")
	}
	return ips, nil
}

// neiAdd add static neighborhood tales
func neiAdd(iface, mac string, ips []string) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}

	// add host neighborhood in pod
	for _, ip := range ips {
		if err := netlink.NeighAdd(&netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			State:        netlink.NUD_PERMANENT,
			Type:         netlink.NDA_LLADDR,
			Flags:        netlink.NTF_SELF,
			IP:           net.ParseIP(ip),
			HardwareAddr: parseMac(mac),
		}); err != nil {
			return err
		}
	}
	return nil
}

// routeAdd add route tables
func routeAdd(iface string, ips []string) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		netIP := net.ParseIP(ip)
		dst := &net.IPNet{
			IP: netIP,
		}
		if netIP.To4() != nil {
			dst.Mask = net.CIDRMask(32, 32)
		} else {
			dst.Mask = net.CIDRMask(128, 128)
		}

		if err = netlink.RouteAdd(&netlink.Route{
			LinkIndex: link.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       dst,
		}); err != nil {
			return err
		}
	}
	return nil
}

// setLinkup set the given interface to up.
func setLinkup(iface string) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}

	// set link up
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set %q UP: %v", iface, err)
	}
	return nil
}

// setRPFilter set rp_filter parameters to 2
func setRPFilter() error {
	dirs, err := os.ReadDir(sysctlConfPath)
	if err != nil {
		return fmt.Errorf("[veth]failed to set rp_filter: %v", err)
	}
	for _, dir := range dirs {
		name := fmt.Sprintf("/net/ipv4/conf/%s/rp_filter", dir.Name())
		value, err := sysctl.Sysctl(name)
		if err != nil {
			continue
		}
		if value == string(rune(RPFilterStrict)) {
			sysctl.Sysctl(name, string(rune(RPFilterLoose)))
		}
	}
	return nil
}

// parseMac parse hardware addr from given string
func parseMac(s string) net.HardwareAddr {
	hardwareAddr, err := net.ParseMAC(s)
	if err != nil {
		panic(err)
	}
	return hardwareAddr
}

// filterIPs filter
// If the same ip family has multiple ip addresses, then only the first one returned.
func filterIPs(netIP net.IP, ipv4, ipv6 bool, viaIps []string) (bool, bool, []string) {
	if netIP.To4() != nil && !ipv4 {
		viaIps = append(viaIps, netIP.String())
		ipv4 = true
	}
	if netIP.To4() == nil && !ipv6 {
		viaIps = append(viaIps, netIP.String())
		ipv6 = true
	}
	return ipv4, ipv6, viaIps
}

// getHostVethName select the first 11 characters of the containerID for the host veth.
func getHostVethName(containerID string) string {
	return fmt.Sprintf("veth%s", containerID[:min(len(containerID))])
}

// isSkipped returns true by checking if the veth0  exists in the container
func isSkipped(netns ns.NetNS) bool {
	skipped := false
	netns.Do(func(_ ns.NetNS) error {
		_, err := netlink.LinkByName(defaultConVeth)
		if err != nil && err == ip.ErrLinkNotFound {
			skipped = true
			return nil
		}
		return nil
	})

	return skipped
}

func min(len int) int {
	if len > 11 {
		return 11
	}
	return len
}

func rpValue(value RPFilterValue) *RPFilterValue {
	return &value
}
