// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"net"
)

var defaultMtu = 1500

// getChainedInterfaceIps return all ip addresses on the NIC of a given netns, including ipv4 and ipv6
func getChainedInterfaceIps(netns ns.NetNS, interfacenName string) ([]string, error) {
	ips := make([]string, 0, 2)
	err := netns.Do(func(_ ns.NetNS) error {
		ipv4, ipv6 := false, false
		ifaces, err := net.Interfaces()
		if err != nil {
			return fmt.Errorf("[veth] failed to list interfaces inside pod")
		}
		for _, iface := range ifaces {
			if iface.Name == interfacenName {
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

// neighborAdd add static neighborhood tales
func neighborAdd(logger *zap.Logger, iface, mac string, ips []string) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("failed to get link: %v", err)
	}

	//  add host neighborhood in pod
	for _, ip := range ips {
		neigh := &netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			State:        netlink.NUD_PERMANENT,
			Type:         netlink.NDA_LLADDR,
			IP:           net.ParseIP(ip),
			HardwareAddr: parseMac(mac),
		}
		if err := netlink.NeighAdd(neigh); err != nil && err.Error() != "file exists" {
			logger.Error("failed to add neigh table", zap.String("interface", iface), zap.String("neigh", neigh.String()), zap.Error(err))
			return fmt.Errorf("failed to add neigh table: %v ", err)
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
/*
var sysctlConfPath = "/proc/sys/net/ipv4/conf"
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
		if value == "1" {
			if _, e := sysctl.Sysctl(name, "2"); e != nil {
				return e
			}
		}
	}
	return nil
}
*/

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

func min(len int) int {
	if len > 11 {
		return 11
	}
	return len
}
