package networking

import (
	"fmt"
	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spidernet-io/cni-plugins/pkg/ipchecking"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
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
