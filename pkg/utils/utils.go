package utils

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"regexp"
	"strings"
)

// var defaultInterfaceName = "eth0"
// var defaultMtu = 1500
// var defaultConVeth = "veth0"
var sysctlConfPath = "/proc/sys/net/ipv4/conf"
var disableIPv6SysctlTemplate = "net/ipv6/conf/%s/disable_ipv6"
var ErrFileExists = "file exists"

var DefaultInterfacesToExclude = []string{
	"docker.*", "cbr.*", "dummy.*",
	"virbr.*", "lxcbr.*", "veth.*", "lo",
	"cali.*", "tunl.*", "flannel.*", "kube-ipvs.*", "cni.*",
}

// GetHostIps return all ip addresses on the node, including ipv4 and ipv6.
// skipping any interfaces whose name matches any of the exclusion list regexes
func GetHostIps() ([]string, error) {
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

// RouteAdd add route tables
func RouteAdd(iface string, ips []string) error {
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
		}); err != nil && err.Error() != ErrFileExists {
			return err
		}
	}
	return nil
}

func IsInSubnet(netIP net.IP, subnet net.IPNet) bool {
	return subnet.Contains(netIP)
}

// SysctlRPFilter set rp_filter value
func SysctlRPFilter(netns ns.NetNS, rp *types.RPFilter ) error {
	var err error
	// set host rp_filter
	if *rp.Enable {
		if err = setRPFilter(*rp.Value); err != nil {
			return err
		}
	}
	// set pod rp_filter
	err = netns.Do(func(_ ns.NetNS) error {
		if err := setRPFilter(*rp.Value); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// setRPFilter set rp_filter parameters
func setRPFilter(v int32) error {
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
			if _, e := sysctl.Sysctl(name, fmt.Sprintf("%d", v)); e != nil {
				return e
			}
		}
	}
	return nil
}
