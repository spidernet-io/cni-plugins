// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
)

var defaultMtu = 1500

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
