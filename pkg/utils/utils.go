package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/spidernet-io/cni-plugins/pkg/constant"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"k8s.io/utils/pointer"
	"math/big"
	"net"
	"net/netip"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var defaultInterfaceName = "eth0"
var overlayRouteTable = 100
var interfacePrefix = "net"

var sysctlConfPathIPv4 = "/proc/sys/net/ipv4/conf"
var sysctlConfPathIPv6 = "/proc/sys/net/ipv6/conf"

var ErrFileExists = "file exists"
var ErrFileNotFound = "no such file or directory"

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
					ips = append(ips, addr.String())
				}
				if ip.To4() == nil && enableIpv6 {
					ips = append(ips, addr.String())
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

// GetChainedInterfaceIps return all ip addresses on the NIC of a given netns, including ipv4 and ipv6
func GetChainedInterfaceIps(netns ns.NetNS, interfacenName string, enableIPv4, enableIpv6 bool) ([]string, error) {
	var err error
	chainedInterfaceIps := make([]string, 0, 2)
	err = netns.Do(func(_ ns.NetNS) error {
		netInterfaces, err := net.Interfaces()
		if err != nil {
			return fmt.Errorf("failed to list container interfaces: %v", err)
		}

		for _, netInterface := range netInterfaces {
			if netInterface.Name == interfacenName {
				addrs, err := netInterface.Addrs()
				if err != nil {
					return fmt.Errorf("failed to list all address for interface %s: %v", netInterface.Name, err)
				}
				for _, addr := range addrs {
					netIP, _, err := net.ParseCIDR(addr.String())
					if err != nil {
						return fmt.Errorf("failed to parse cidr %s: %v", addr.String(), err)
					}
					if netIP.IsMulticast() || netIP.IsLinkLocalUnicast() {
						continue
					}
					if netIP.To4() != nil && enableIPv4 {
						chainedInterfaceIps = append(chainedInterfaceIps, addr.String())
					}
					if netIP.To4() == nil && enableIpv6 {
						chainedInterfaceIps = append(chainedInterfaceIps, addr.String())
					}
				}
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(chainedInterfaceIps) == 0 {
		return nil, fmt.Errorf("no one vaild ip on the pod")
	}
	return chainedInterfaceIps, nil
}

// RouteAdd add route tables from given ips and iface
// Equivalent: ip route add <dst> device <iface>
func RouteAdd(logger *zap.Logger, ruleTable int, iface string, ips []string, enableIpv4, enableIpv6 bool) (dst4 *net.IPNet, dst6 *net.IPNet, e error) {
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
		netIP, _, err := net.ParseCIDR(ip)
		if err != nil {
			logger.Error(err.Error())
			return nil, nil, err
		}
		dst := &net.IPNet{
			IP: netIP,
		}
		if netIP.To4() != nil && enableIpv4 {
			dst.Mask = net.CIDRMask(32, 32)
			dst4 = dst
		}
		if netIP.To4() == nil && enableIpv6 {
			dst.Mask = net.CIDRMask(128, 128)
			dst6 = dst
		}

		if dst.Mask == nil {
			logger.Warn("IP is don't match with the IP-Family", zap.String("IP", netIP.String()))
			continue
		}

		if err = netlink.RouteAdd(&netlink.Route{
			LinkIndex: link.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       dst,
			Table:     ruleTable,
		}); err != nil && err.Error() != constant.ErrFileExists {
			logger.Error("failed to add route", zap.String("interface", iface), zap.String("dst", dst.IP.String()), zap.Error(err))
			return nil, nil, err
		}
	}
	return dst4, dst6, nil
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
		v = pointer.Int32(0)
	}
	logger.Debug("Setting rp_filter", zap.Int32("value", *v))
	dirs, err := os.ReadDir(sysctlConfPathIPv4)
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
		if value == fmt.Sprintf("%d", *v) {
			continue
		}
		if _, e := sysctl.Sysctl(name, fmt.Sprintf("%d", *v)); e != nil {
			logger.Error("failed to set rp_filter", zap.String("name", name), zap.Error(err))
			return e
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

func EnableIpv6Sysctl(logger *zap.Logger, netns ns.NetNS) error {
	logger.Debug("Setting all interface sysctl 'disable_ipv6' to 0 ", zap.String("NetNs Path", netns.Path()))
	err := netns.Do(func(_ ns.NetNS) error {
		dirs, err := os.ReadDir(sysctlConfPathIPv6)
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		for _, dir := range dirs {
			// Read current sysctl value
			name := fmt.Sprintf("/net/ipv6/conf/%s/disable_ipv6", dir.Name())
			value, err := sysctl.Sysctl(name)
			if err != nil {
				logger.Error("failed to read current sysctl value", zap.String("name", name), zap.Error(err))
				return fmt.Errorf("failed to read current sysctl %+v value: %v", name, err)
			}
			// make sure value=0
			if value != "0" {
				if _, err = sysctl.Sysctl(name, "0"); err != nil {
					logger.Error("failed to set sysctl value to 0 ", zap.String("name", name), zap.Error(err))
					return fmt.Errorf("failed to read current sysctl %+v value: %v ", name, err)
				}
			}
		}
		return nil
	})
	return err
}

// HijackCustomSubnet set ip rule : to Subnet table $routeTable
// if first macvlan interface, move service/pod subnet route to table 100: ip rule add from all to service/pod look table 100
// else only move custom route to table <ruleTable>: ip rule add from all to <custom_subnet> look table <ruletable>
func HijackCustomSubnet(logger *zap.Logger, netns ns.NetNS, serviceSubnet, overlaySubnet, additionalSubnet []string, defaultInterfaceIPs []netlink.Addr, routeTable int, enableIpv4, enableIpv6 bool) error {
	logger.Debug(fmt.Sprintf("Hijack Custom Subnet to %v ", routeTable), zap.String("Netns Path", netns.Path()),
		zap.Bool("enableIpv4", enableIpv4),
		zap.Bool("enableIpv6", enableIpv6))
	e := netns.Do(func(_ ns.NetNS) error {
		var err error
		// only first macvlan interface, we add rule table for it.
		// eq: ip rule add from <overlay/service subnet> lookup <ruleTable>
		if routeTable == overlayRouteTable {
			allSubnets := append(overlaySubnet, serviceSubnet...)
			if err = ruleAdd(logger, allSubnets, routeTable, enableIpv4, enableIpv6); err != nil {
				return err
			}

		} else {
			// As for more than two macvlan interface, we need to add something like below shown:
			// eq: ip rule add to <defaultInterfaceIPs > lookup table <ruleTable>
			// net2: ip rule add to <net1 subnet> lookup table <ruleTable>
			if err = toRuleAdd(logger, defaultInterfaceIPs, routeTable, enableIpv4, enableIpv6); err != nil {
				return err
			}
		}

		// last we hijack additionalSubnet to lookup table <routeTable>
		if err = ruleAdd(logger, additionalSubnet, routeTable, enableIpv4, enableIpv6); err != nil {
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
			logger.Error(err.Error())
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
		if err := netlink.RuleAdd(rule); err != nil && !strings.EqualFold(err.Error(), constant.ErrRouteFileExist) {
			logger.Error(err.Error())
			return fmt.Errorf("failed to set ip rule(%+v rule.Dst %v): %+v", rule, rule.Dst, err)
		}
	}
	return nil
}

func toRuleAdd(logger *zap.Logger, routes []netlink.Addr, routeTable int, enableIpv4, enableIpv6 bool) error {
	for _, route := range routes {
		var family int
		match := false
		if route.IP.To4() != nil && enableIpv4 {
			family = netlink.FAMILY_V4
			match = true
		}
		if route.IP.To4() == nil && enableIpv6 {
			family = netlink.FAMILY_V6
			match = true
		}

		if !match {
			logger.Warn("the route given does not match the ipVersion of the pod, ignore the creation of this route", zap.String("route", route.String()))
			continue
		}

		rule := netlink.NewRule()
		rule.Dst = route.IPNet
		rule.Family = family
		rule.Table = routeTable
		logger.Debug("HijackCustomSubnet Add Rule table", zap.Int("ipfamily", family), zap.String("dst", rule.Dst.String()))
		if err := netlink.RuleAdd(rule); err != nil && !strings.EqualFold(err.Error(), constant.ErrRouteFileExist) {
			logger.Error(err.Error())
			return fmt.Errorf("failed to set ip rule(%+v rule.Dst %v): %+v", rule, rule.Dst, err)
		}
	}
	return nil
}

// MigrateRoute make sure that the reply packets accessing the overlay interface are still sent from the overlay interface.
func MigrateRoute(logger *zap.Logger, netns ns.NetNS, defaultInterface, chainedInterface string, defaultInterfaceIPs []netlink.Addr, value types.MigrateRoute, ruleTable int, enableIpv4, enableIpv6 bool) error {
	/*
		1. if migrateValue = -1, auto migrate route by interface name, if current_interface > last_interface by directory order, do migrate else nothing to do
		2. if migrateValue = 1, do migrate directly
		3. do migrate:
		 		a. add rule table by given interface name: ip rule add from <interface>/32 lookup table <ruleTable>
				b. move all route of given defaultInterface to table 100
		4. end
	*/
	if value == types.MigrateNever {
		return nil
	}

	var err error
	var defaultRouteInterface string
	if value == types.MigrateAuto {
		if !compareInterfaceName(chainedInterface, defaultInterfaceName) {
			logger.Info("Ignore migrate default route", zap.String("Current Interface", defaultInterface), zap.String("Default Route Interface", defaultRouteInterface))
			return nil
		}
	}

	logger.Debug("hijack overlay response packet to overlay interface",
		zap.String("defaultRouteString", defaultInterface),
		zap.Int32("migrate_route", int32(value)),
		zap.Bool("enableIpv4", enableIpv4),
		zap.Bool("enableIpv6", enableIpv6))
	// add route rule: source overlayIP for new rule
	// eq: ip rule add from <defaultRoute interface> lookup <ruleTable>
	logger.Debug("Add Rule Table in Pod Netns", zap.Int("ruleTable", ruleTable), zap.Any("chainedIPs", defaultInterfaceIPs))
	err = netns.Do(func(_ ns.NetNS) error {
		if err = AddFromRuleTable(logger, defaultInterfaceIPs, ruleTable, enableIpv4, enableIpv6); err != nil {
			logger.Error(fmt.Sprintf("failed to add route table %d: %v ", overlayRouteTable, err))
			return err
		}

		return nil
	})

	// move overlay default route to table <ruleTable>
	if enableIpv4 {
		err := netns.Do(func(_ ns.NetNS) error {
			return moveRouteTable(logger, defaultInterface, ruleTable, netlink.FAMILY_V4)
		})
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}
	if enableIpv6 {
		err := netns.Do(func(_ ns.NetNS) error {
			return moveRouteTable(logger, defaultInterface, ruleTable, netlink.FAMILY_V6)
		})
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}
	return nil
}

// AddFromRuleTable add route rule for calico/cilium cidr(ipv4 and ipv6)
// Equivalent to: `ip rule add from <cidr> `
func AddFromRuleTable(logger *zap.Logger, chainedIPs []netlink.Addr, ruleTable int, enableIpv4, enableIpv6 bool) error {
	logger.Debug("Add FromRule Table in Pod Netns")
	for _, chainedIP := range chainedIPs {
		mask := net.IPMask{}
		if chainedIP.IP.To4() != nil && enableIpv4 {
			mask = net.CIDRMask(32, 32)
		}
		if chainedIP.IP.To4() == nil && enableIpv6 {
			mask = net.CIDRMask(128, 128)
		}

		rule := netlink.NewRule()
		rule.Table = ruleTable
		rule.Src = &net.IPNet{
			IP:   chainedIP.IP,
			Mask: mask,
		}
		logger.Debug("Netlink RuleAdd", zap.String("Rule", rule.String()))
		if err := netlink.RuleAdd(rule); err != nil {
			logger.Error(err.Error())
			return err
		}
	}
	// we should add rule route table, just like `ip route add default via 169.254.1.1 table 100`
	// but we don't know what's the default route If it has been deleted.
	// so we should add this route rule table before removing the default route
	return nil
}

func AddrListByName(iface string, family int) ([]netlink.Addr, error) {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return nil, err
	}

	addrs, err := netlink.AddrList(link, family)
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

// AddToRuleTable
func AddToRuleTable(logger *zap.Logger, chainedIPs []string, ruleTable int, enableIpv4, enableIpv6 bool) error {
	for _, chainedIP := range chainedIPs {
		netIP, ipNet, err := net.ParseCIDR(chainedIP)
		if err != nil {
			return fmt.Errorf("failed to parse cidr %s: %v", chainedIP, err)
		}
		if netIP.IsMulticast() || netIP.IsLinkLocalUnicast() {
			continue
		}

		rule := netlink.NewRule()
		rule.Table = ruleTable
		rule.Dst = ipNet
		logger.Debug("Netlink RuleAdd", zap.String("Rule", rule.String()))
		if err = netlink.RuleAdd(rule); err != nil {
			logger.Error(err.Error())
			return err
		}
	}
	// we should add rule route table, just like `ip route add default via 169.254.1.1 table 100`
	// but we don't know what's the default route If it has been deleted.
	// so we should add this route rule table before removing the default route
	return nil
}

// moveRouteTable del default route and add default rule route in pod netns
// Equivalent: `ip route del <default route>` and `ip r route add <default route> table 100`
func moveRouteTable(logger *zap.Logger, iface string, ruleTable, ipfamily int) error {
	logger.Debug(fmt.Sprintf("Moving overlay route table from main table to %d by given interface", ruleTable),
		zap.String("interface", iface),
		zap.Int("ipfamily", ipfamily))
	link, err := netlink.LinkByName(iface)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	routes, err := netlink.RouteList(nil, ipfamily)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	for _, route := range routes {

		// only handle route tables from table main
		if route.Table != unix.RT_TABLE_MAIN {
			continue
		}

		// ingore local link route
		if route.Dst.String() == "fe80::/64" {
			continue
		}

		logger.Debug("Found Route", zap.String("Route", route.String()))

		if route.LinkIndex == link.Attrs().Index {
			if route.Dst == nil {
				if err = netlink.RouteDel(&route); err != nil {
					logger.Error("failed to delete default route  in main table ", zap.String("route", route.String()), zap.Error(err))
					return fmt.Errorf("failed to delete default route (%+v) in main table: %+v", route, err)
				}
				logger.Debug("Succeed to del the default route", zap.String("Default Route", route.String()))
			}
			route.Table = ruleTable
			if err = netlink.RouteAdd(&route); err != nil && err.Error() != constant.ErrRouteFileExist {
				logger.Error("failed to add default route to new table ", zap.String("route", route.String()), zap.Error(err))
				return fmt.Errorf("failed to add route (%+v) to new table: %+v", route, err)
			}
			logger.Debug("Succeed to move default route table from main to new table", zap.String("Route", route.String()))
		} else {
			// especially more than two default ipv6 gateway
			var generatedRoute, deletedRoute *netlink.Route
			if len(route.MultiPath) == 0 {
				continue
			}

			// get generated default Route for new table
			for _, v := range route.MultiPath {
				if v.LinkIndex == link.Attrs().Index {
					generatedRoute = &netlink.Route{
						LinkIndex: v.LinkIndex,
						Gw:        v.Gw,
						Table:     ruleTable,
						MTU:       route.MTU,
					}
					deletedRoute = &netlink.Route{
						LinkIndex: v.LinkIndex,
						Gw:        v.Gw,
						Table:     unix.RT_TABLE_MAIN,
					}
					break
				}
			}
			if generatedRoute == nil {
				continue
			}
			// add default route to new table
			if err = netlink.RouteAdd(generatedRoute); err != nil && err.Error() != constant.ErrRouteFileExist {
				logger.Error("failed to add overlay route to new table", zap.String("generatedRoute", generatedRoute.String()), zap.Error(err))
				return fmt.Errorf("failed to move overlay route (%+v) to new table: %+v", generatedRoute, err)
			}
			// delete default route in main table
			if err := netlink.RouteDel(deletedRoute); err != nil {
				logger.Error("failed to del overlay route from main table", zap.String("deletedRoute", deletedRoute.String()), zap.Error(err))
				return fmt.Errorf("failed to delete default route (%+v) in main table: %+v", deletedRoute, err)
			}
		}
	}
	return nil
}

// GetRuleNumber return the number of rule table corresponding to the previous interface from the given interface.
// the input format must be 'net+number'
// for example:
// input: net1, output: 100(eth0)
// input: net2, output: 101(net1)
func GetRuleNumber(iface string) int {
	if !strings.HasPrefix(iface, "net") {
		return -1
	}
	numStr := strings.Trim(iface, "net")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}
	return overlayRouteTable + num - 1
}

// GetDefaultRouteInterface return last interface from given iface.
// example:
// input: net1, output: eth0
// input: net2, output: net1
// ...
func GetDefaultRouteInterface(iface string) string {
	if iface == "net1" {
		return defaultInterfaceName
	}
	numStr := strings.Trim(iface, "net")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%d", "net", num-1)
}

// compareInterfaceName compare name from given current and prev by directory order
// example:
// net1 > eth0, true
// net2 > net1, true
func compareInterfaceName(current, prev string) bool {
	if prev == defaultInterfaceName {
		return true
	}

	if !strings.HasPrefix(current, interfacePrefix) || !strings.HasPrefix(prev, interfacePrefix) {
		return false
	}
	return current >= prev
}

func GetNextHopIPs(logger *zap.Logger, ips []string) ([]net.IP, error) {
	viaIPs := make([]net.IP, 0, 2)
	for _, nip := range ips {
		netIP, _, err := net.ParseCIDR(nip)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cidr %s: %v", nip, err)
		}
		logger.Debug("destination IP", zap.Any("dst", netIP))
		routes, err := netlink.RouteGet(netIP)
		if err != nil {
			return nil, fmt.Errorf("failed to ip route get %s: %v", nip, err)
		}

		for _, route := range routes {
			viaIPs = append(viaIPs, route.Src)
			break
		}

	}
	if len(viaIPs) == 0 {
		return nil, fmt.Errorf("nexthoop ips no found on host")
	}
	return viaIPs, nil
}

func RuleDel(logger *zap.Logger, ruleTable int, ips []netlink.Addr) error {
	logger.Debug("Del Rule Table", zap.Int("RuleTable", ruleTable), zap.Any("ChainedInterface IP", ips))

	for _, chainedIP := range ips {
		dst := net.IPNet{
			IP:   chainedIP.IP,
			Mask: net.IPMask{},
		}

		if chainedIP.IP.To4() != nil {
			dst.Mask = net.CIDRMask(32, 32)
		} else {
			dst.Mask = net.CIDRMask(128, 128)
		}

		rule := netlink.NewRule()
		rule.Table = ruleTable
		rule.Dst = &dst
		if err := netlink.RuleDel(rule); err != nil && !os.IsNotExist(err) {
			logger.Error("failed to del rule table", zap.Error(err))
			return fmt.Errorf("failed to del rule table %d: %v ", ruleTable, err)
		}
	}

	return nil
}

// AddStaticNeighTable fix the problem of communication failure between pods and hosts by adding neigh table on pod and host
func AddStaticNeighTable(logger *zap.Logger, netns ns.NetNS, iSriov bool, defaultOverlayInterface string, hostIPs []net.IP, chainedInterfaceIps []netlink.Addr) error {
	if iSriov {
		logger.Info("Main-cni is sriov, don't need set chained route")
		return nil
	}

	parentIndex := -1
	defaultOverlayMac := ""
	err := netns.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(defaultOverlayInterface)
		if err != nil {
			logger.Error("", zap.Error(err))
			return err
		}
		// get link index of host veth-peer and pod veth-peer mac-address
		parentIndex = link.Attrs().ParentIndex
		defaultOverlayMac = link.Attrs().HardwareAddr.String()
		return nil
	})
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	if parentIndex < 0 {
		logger.Debug("defaultOverlay veth-peer linkIndex no found, ignore add neigh table")
		return nil
	}

	if defaultOverlayMac == "" {
		logger.Debug("defaultOverlayInterface Mac-address still empty, ignore add neigh table")
		return nil
	}

	hostLink, err := netlink.LinkByIndex(parentIndex)
	if err != nil {
		logger.Error("", zap.Error(err))
		return err
	}

	// add neigh table in pod
	// eq: ip n add <host IP> dev eth0 lladdr <host veth-peer mac> nud permanent
	err = netns.Do(func(_ ns.NetNS) error {
		for _, hostIP := range hostIPs {
			if err := NeighborAdd(logger, defaultOverlayInterface, hostLink.Attrs().HardwareAddr.String(), hostIP); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	// eq: ip n add <chained interface IP> dev <host veth-peer > lladdr < defaultInterface Mac> nud permanent (only for ipv6)
	for _, chainedInterfaceIP := range chainedInterfaceIps {
		if err = NeighborAdd(logger, hostLink.Attrs().Name, defaultOverlayMac, chainedInterfaceIP.IP); err != nil {
			logger.Error(err.Error())
			return err
		}
	}
	return nil
}

// NeighborAdd add static neighborhood tales
func NeighborAdd(logger *zap.Logger, iface, mac string, netIP net.IP) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("failed to get link: %v", err)
	}

	neigh := &netlink.Neigh{
		LinkIndex:    link.Attrs().Index,
		State:        netlink.NUD_PERMANENT,
		Type:         netlink.NDA_LLADDR,
		IP:           netIP,
		HardwareAddr: parseMac(mac),
	}

	if err := netlink.NeighAdd(neigh); err != nil && !os.IsExist(err) {
		logger.Error("failed to add neigh table", zap.String("interface", iface), zap.String("neigh", neigh.String()), zap.Error(err))
		return fmt.Errorf("failed to add neigh table(%+v): %v ", neigh, err)
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

// OverwriteMacAddress overwrite mac-address
func OverwriteMacAddress(logger *zap.Logger, netns ns.NetNS, macPrefix, iface string) (string, error) {
	// which nic need to overwrite?
	logger.Debug("Get OverwriteMacAddress parameters", zap.String("macPrefix", macPrefix), zap.String("iface", iface))
	ips, err := GetChainedInterfaceIps(netns, iface, true, true)
	if err != nil {
		logger.Error("failed to get ips from given interface inside pod", zap.String("interface", iface), zap.Error(err))
		return "", err
	}

	// in fact, we only focus on first ip in ips
	nAddr, err := netip.ParsePrefix(ips[0])
	if err != nil {
		logger.Error("failed to parse ip to addr", zap.Error(err))
		return "", err
	}

	suffix, err := inetAton(nAddr.Addr())
	if err != nil {
		logger.Error("failed to get suffix of mac address", zap.Error(err))
		return "", err
	}

	// newmac = xx:xx + xx:xx:xx:xx
	newMac := macPrefix + ":" + suffix
	err = netns.Do(func(netNS ns.NetNS) error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		return netlink.LinkSetHardwareAddr(link, parseMac(newMac))
	})

	if err != nil {
		logger.Error("failed to overwrite mac address", zap.String("newMac", newMac), zap.Error(err))
		return "", err
	}
	return newMac, nil
}

// inetAton converts an IP Address (IPv4 or IPv6) netip.addr object to a hexadecimal representation
func inetAton(ip netip.Addr) (string, error) {
	if ip.AsSlice() == nil {
		return "", fmt.Errorf("invalid ip address")
	}

	ipInt := big.NewInt(0)
	// 32 bit -> 4 B
	hexCode := make([]byte, hex.EncodedLen(ip.BitLen()))
	ipBytes := ip.AsSlice()
	ipInt.SetBytes(ipBytes[:])
	hex.Encode(hexCode, ipInt.Bytes())

	if ip.Is6() {
		// for ipv6: 128 bit = 32 hex
		// take the last 8 hex as the mac address
		return convertHex2Mac(hexCode[24:]), nil
	}

	return convertHex2Mac(hexCode), nil
}

// convertHex2Mac convert hexcode to 8bit mac
func convertHex2Mac(hexCode []byte) string {
	// convert ip(hex) to "xx:xx:xx:xx"
	// group by 2bit
	regexSpilt, err := regexp.Compile(".{2}")
	if err != nil {
		panic(err)
	}
	return string(bytes.Join(regexSpilt.FindAll(hexCode, 4), []byte(":")))
}
