package utils

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"k8s.io/utils/pointer"
	"net"
	"os"
	"regexp"
	"strings"
)

// var defaultInterfaceName = "eth0"
// var defaultMtu = 1500
// var defaultConVeth = "veth0"
var sysctlConfPath = "/proc/sys/net/ipv4/conf"

// var disableIPv6SysctlTemplate = "net/ipv6/conf/%s/disable_ipv6"
var ErrFileExists = "file exists"

var DefaultInterfacesToExclude = []string{
	"docker.*", "cbr.*", "dummy.*",
	"virbr.*", "lxcbr.*", "veth.*", "lo",
	"cali.*", "tunl.*", "flannel.*", "kube-ipvs.*", "cni.*",
}

// GetHostIps return all ip addresses on the node, filter by ipFamily
// skipping any interfaces whose name matches any of the exclusion list regexes
func GetHostIps(logger *zap.Logger, enableIPv4, enableIpv6 bool) ([]string, error) {
	netIfaces, err := net.Interfaces()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	var ips []string
	var excludeRegexp *regexp.Regexp

	if excludeRegexp, err = regexp.Compile("(" + strings.Join(DefaultInterfacesToExclude, ")|(") + ")"); err != nil {
		logger.Error(err.Error())
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
				if ip.To4() != nil && enableIPv4 {
					ips = append(ips, ip.String())
				}
				if ip.To4() == nil && enableIpv6 {
					ips = append(ips, ip.String())
				}
			}
		}
	}

	if len(ips) == 0 {
		logger.Error("no one vaild ip on the node")
		return nil, fmt.Errorf("no one vaild ip on the node")
	}
	logger.Debug("Get the ip addresses of all interfaces on the node", zap.Strings("Host IPs", ips))
	return ips, nil
}

// RouteAdd add route tables from given ips and iface
// Equivalent: ip route add <dst> device <iface>
func RouteAdd(logger *zap.Logger, iface string, ips []string, enableIpv4, enableIpv6 bool) (dst4 *net.IPNet, dst6 *net.IPNet, e error) {
	logger.Debug("Add route table", zap.String("Device", iface),
		zap.Strings("Destinations", ips),
		zap.Bool("enableIpv4", enableIpv4),
		zap.Bool("enableIpv6", enableIpv6))
	link, err := netlink.LinkByName(iface)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil, err
	}
	for _, ip := range ips {
		netIP := net.ParseIP(ip)
		dst := &net.IPNet{
			IP: netIP,
		}
		if netIP.To4() != nil {
			if !enableIpv4 {
				continue
			}
			dst.Mask = net.CIDRMask(32, 32)
			dst4 = dst
		} else {
			if !enableIpv6 {
				continue
			}
			dst.Mask = net.CIDRMask(128, 128)
			dst6 = dst
		}

		if err = netlink.RouteAdd(&netlink.Route{
			LinkIndex: link.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       dst,
		}); err != nil && err.Error() != ErrFileExists {
			logger.Error("failed to add route", zap.String("interface", iface), zap.String("dst", dst.String()), zap.Error(err))
			return nil, nil, err
		}
	}
	return dst4, dst6, nil
}

func IsInSubnet(netIP net.IP, subnet net.IPNet) bool {
	return subnet.Contains(netIP)
}

// SysctlRPFilter set rp_filter value
func SysctlRPFilter(logger *zap.Logger, netns ns.NetNS, rp *types.RPFilter) error {
	var err error
	if rp.Enable != nil && *rp.Enable {
		if err = setRPFilter(logger, rp.Value); err != nil {
			logger.Error(fmt.Sprintf("failed to set rp_filter for host : %v", err))
			return fmt.Errorf("failed to set rp_filter for host : %v", err)
		}
	}
	// set pod rp_filter
	err = netns.Do(func(_ ns.NetNS) error {
		if err := setRPFilter(logger, rp.Value); err != nil {
			logger.Error(fmt.Sprintf("failed to set rp_filter for pod : %v", err))
			return fmt.Errorf("failed to set rp_filter for pod : %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// setRPFilter set rp_filter parameters
func setRPFilter(logger *zap.Logger, v *int32) error {
	if v == nil {
		v = pointer.Int32(2)
	}
	logger.Debug("Setting rp_filter", zap.Int32("value", *v))
	dirs, err := os.ReadDir(sysctlConfPath)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	for _, dir := range dirs {
		name := fmt.Sprintf("/net/ipv4/conf/%s/rp_filter", dir.Name())
		value, err := sysctl.Sysctl(name)
		if err != nil {
			logger.Warn("failed to get rp_filter value", zap.String("name", name), zap.Error(err))
			continue
		}
		if value == "1" {
			if _, e := sysctl.Sysctl(name, fmt.Sprintf("%d", *v)); e != nil {
				logger.Error("failed to set rp_filter", zap.String("name", name), zap.Error(err))
				return e
			}
		}
	}
	return nil
}

// CheckInterfaceMiss returns true by checking if the veth0  exists in the container
func CheckInterfaceMiss(netns ns.NetNS, intefaceName string) (bool, error) {
	e := netns.Do(func(_ ns.NetNS) error {
		_, err := netlink.LinkByName(intefaceName)
		return err
	})

	if e == nil {
		return false, nil
	}
	if strings.EqualFold(e.Error(), ip.ErrLinkNotFound.Error()) {
		return true, nil
	} else {
		return false, e
	}
}

func EnableIpv6Sysctl(logger *zap.Logger, netns ns.NetNS, DefaultOverlayInterface string) error {
	logger.Debug("Setting container sysctl 'disable_ipv6' to 0 ", zap.String("NetNs Path", netns.Path()),
		zap.String("OverlayInterface", DefaultOverlayInterface))
	err := netns.Do(func(_ ns.NetNS) error {
		allConf := []string{
			fmt.Sprintf("net/ipv6/conf/%s/disable_ipv6", DefaultOverlayInterface),
			"net/ipv6/conf/all/disable_ipv6",
		}
		for _, v := range allConf {
			// Read current sysctl value
			value, err := sysctl.Sysctl(v)
			if err != nil {
				logger.Error("failed to read current sysctl value", zap.String("name", v), zap.Error(err))
				return fmt.Errorf("failed to read current sysctl %+v value: %v", v, err)
			}
			// make sure value=1
			if value != "0" {
				if _, err = sysctl.Sysctl(v, "0"); err != nil {
					logger.Error("failed to set sysctl value to 0 ", zap.String("name", v), zap.Error(err))
					return fmt.Errorf("failed to read current sysctl %+v value: %v ", v, err)
				}
			}
		}
		return nil
	})
	return err
}

// HijackCustomSubnet set ip rule : to Subnet table $routeTable
func HijackCustomSubnet(logger *zap.Logger, netns ns.NetNS, routes *types.Route, routeTable int, enableIpv4, enableIpv6 bool) error {
	logger.Debug(fmt.Sprintf("Hijack Custom Subnet to %v ", routeTable), zap.String("Netns Path", netns.Path()),
		zap.Bool("enableIpv4", enableIpv4),
		zap.Bool("enableIpv6", enableIpv6))
	e := netns.Do(func(_ ns.NetNS) error {
		var err error
		if err = ruleAdd(logger, routes.OverlaySubnet, routeTable, enableIpv4, enableIpv6); err != nil {
			return err
		}
		if err = ruleAdd(logger, routes.ServiceSubnet, routeTable, enableIpv4, enableIpv6); err != nil {
			return err
		}
		if err = ruleAdd(logger, routes.CustomSubnet, routeTable, enableIpv4, enableIpv6); err != nil {
			return err
		}
		return nil
	})
	return e
}

// ruleAdd
func ruleAdd(logger *zap.Logger, routes []string, routeTable int, enableIpv4, enableIpv6 bool) error {
	for _, route := range routes {
		_, ipNet, err := net.ParseCIDR(route)
		if err != nil {
			return err
		}
		var family int
		match := false
		if ipNet.IP.To4() != nil && enableIpv4 {
			family = netlink.FAMILY_V4
			match = true
		}
		if ipNet.IP.To4() == nil && enableIpv6 {
			family = netlink.FAMILY_V6
			match = true
		}

		if !match {
			logger.Warn("the route given does not match the ipVersion of the pod, ignore the creation of this route", zap.String("route", ipNet.String()))
			continue
		}

		rule := netlink.NewRule()
		rule.Dst = ipNet
		rule.Family = family
		rule.Table = routeTable
		logger.Debug("HijackCustomSubnet Add Rule table", zap.Int("ipfamily", family), zap.String("dst", rule.Dst.String()))
		if err := netlink.RuleAdd(rule); err != nil {
			logger.Error(err.Error())
			return fmt.Errorf("failed to set ip rule(%+v rule.Dst %v): %+v", rule, rule.Dst, err)
		}
	}
	return nil
}
