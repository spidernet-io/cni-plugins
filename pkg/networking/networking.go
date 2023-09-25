package networking

import (
	"fmt"
	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spidernet-io/cni-plugins/pkg/ipchecking"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/spiderpool/pkg/networking/networking"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"net"
	"net/netip"
	"time"
)

func DoIPConflictChecking(logger *zap.Logger, netns ns.NetNS, iface string, ipconfigs []*types100.IPConfig, config *ty.IPConflict) error {
	logger.Debug("DoIPConflictChecking")

	if len(ipconfigs) == 0 {
		return fmt.Errorf("interface %s has no any ip configured", iface)
	}

	duration, err := time.ParseDuration(config.Interval)
	if err != nil {
		return fmt.Errorf("failed to parse interval %v: %v", config.Interval, err)
	}

	return netns.Do(func(netNS ns.NetNS) error {
		ifi, err := net.InterfaceByName(iface)
		if err != nil {
			return fmt.Errorf("failed to get interface by name %s: %v", iface, err)
		}

		for idx, _ := range ipconfigs {
			target := netip.MustParseAddr(ipconfigs[idx].Address.IP.String())
			if target.Is4() {
				logger.Debug("IPCheckingByARP", zap.String("address", target.String()))
				err = ipchecking.IPCheckingByARP(ifi, target, config.Retry, duration)
				if err != nil {
					return err
				}
				logger.Debug("No IPv4 address conflicting", zap.String("address", target.String()))
			} else {
				logger.Debug("IPCheckingByNDP", zap.String("address", target.String()))
				err = ipchecking.IPCheckingByNDP(ifi, target, config.Retry, duration)
				if err != nil {
					return err
				}
				logger.Debug("No IPv6 address conflicting", zap.String("address", target.String()))
			}
		}
		return nil
	})
}

func durationStr(interval float64) string {
	return fmt.Sprintf("%vs", interval)
}

func GetAllHostIPRouteForPod(ipFamily int, allPodIp []netlink.Addr) (finalNodeIpList []net.IP, e error) {

	finalNodeIpList = []net.IP{}

OUTER1:
	// get node ip by `ip r get podIP`
	for _, item := range allPodIp {
		var t net.IP
		v4Gw, v6Gw, err := networking.GetGatewayIP([]netlink.Addr{item})
		if err != nil {
			return nil, fmt.Errorf("failed to GetGatewayIP for pod ip %+v : %+v ", item, zap.Error(err))
		}
		if len(v4Gw) > 0 && (ipFamily == netlink.FAMILY_V4 || ipFamily == netlink.FAMILY_ALL) {
			t = v4Gw
		} else if len(v6Gw) > 0 && (ipFamily == netlink.FAMILY_V6 || ipFamily == netlink.FAMILY_ALL) {
			t = v6Gw
		}
		for _, k := range finalNodeIpList {
			if k.Equal(t) {
				continue OUTER1
			}
		}
		finalNodeIpList = append(finalNodeIpList, t)
	}

	var DefaultNodeInterfacesToExclude = []string{
		"docker.*", "cbr.*", "dummy.*",
		"virbr.*", "lxcbr.*", "veth.*", `^lo$`,
		`^cali.*`, "flannel.*", "kube-ipvs.*",
		"cni.*", "vx-submariner", "cilium*",
	}

	// get additional host ip
	additionalIp, err := networking.GetAllIPAddress(ipFamily, DefaultNodeInterfacesToExclude)
	if err != nil {
		return nil, fmt.Errorf("failed to get IPAddressOnNode: %v", err)
	}

OUTER2:
	for _, t := range additionalIp {
		if len(t.IP) == 0 {
			continue OUTER2
		}

		for _, k := range finalNodeIpList {
			if k.Equal(t.IP) {
				continue OUTER2
			}
		}
		if t.IP.To4() != nil {
			if ipFamily == netlink.FAMILY_V4 || ipFamily == netlink.FAMILY_ALL {
				finalNodeIpList = append(finalNodeIpList, t.IP)
			}
		} else {
			if ipFamily == netlink.FAMILY_V6 || ipFamily == netlink.FAMILY_ALL {
				finalNodeIpList = append(finalNodeIpList, t.IP)
			}
		}
	}

	return finalNodeIpList, nil
}
