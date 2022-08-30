package main

import (
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	ty "github.com/spidernet-io/veth-plugin/pkg/types"
	"github.com/spidernet-io/veth-plugin/pkg/utils"
	"github.com/vishvananda/netlink"
	"k8s.io/utils/pointer"
)

type PluginConf struct {
	types.NetConf
	Routes []*types.Route `json:"routes,omitempty"`

	// RpFilter
	DelDefaultRoute4        *bool  `json:"delDefaultRoute4,omitempty"`
	DelDefaultRoute6        *bool  `json:"delDefaultRoute6,omitempty"`
	DefaultOverlayInterface string `json:"default_overlay_interface,omitempty"`
	// RpFilter
	RPFilter *ty.RPFilter `json:"rp_filter,omitempty"`
	Skipped  bool         `json:"skip_call,omitempty"`
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("veth"))
}

func cmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	// skip veth plugin
	if conf.Skipped {
		return types.PrintResult(conf.PrevResult, conf.CNIVersion)
	}
	if conf.PrevResult == nil {
		return fmt.Errorf("must be called as chained plugin")
	}

	prevResult, err := current.GetResult(conf.PrevResult)
	if err != nil {
		return fmt.Errorf("failed to convert prevResult: %v", err)
	}

	if len(prevResult.IPs) == 0 {
		return fmt.Errorf("got no container IPs")
	}

	// Pass the prevResult through this plugin to the next one
	// result := prevResult

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	// Add route table
	if err := addRoute(netns, conf); err != nil {
		return fmt.Errorf("[router] failed to add route: %v", err)
	}
	// Add rule route table
	if err := addRouteRule(netns, conf); err != nil {
		return fmt.Errorf("[router] failed to add route rule: %v", err)
	}

	// move default cni's default route to table 100
	if err := movDefaultRoute(netns, conf); err != nil {
		return fmt.Errorf("[router] failed to delete default route for %s: %v", conf.DefaultOverlayInterface, err)
	}

	// 4. setup sysctl rp_filter
	if err = utils.SysctlRPFilter(netns, conf.RPFilter); err != nil {
		return err
	}
	return types.PrintResult(conf.PrevResult, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	// nothing to do
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	// TODO
	return fmt.Errorf("not implement it")
}

// parseConfig parses the supplied configuration (and prevResult) from stdin.
func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := PluginConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("[router] failed to parse network configuration: %v", err)
	}

	// Parse previous result. This will parse, validate, and place the
	// previous result object into conf.PrevResult. If you need to modify
	// or inspect the PrevResult you will need to convert it to a concrete
	// versioned Result struct.
	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("[router] could not parse prevResult: %v", err)
	}
	// End previous result parsing

	// some validation
	for idx, route := range conf.Routes {
		if route.Dst.IP == nil {
			return nil, fmt.Errorf("[router] routes[%v]: des must be specified", idx)
		}
	}

	if conf.DefaultOverlayInterface == "" {
		conf.DefaultOverlayInterface = "eth0"
	}
	// value must be 0/1/2
	// If not, giving default value: RPFilter_Loose(2) to it
	if conf.RPFilter != nil {
		if conf.RPFilter.Enable != nil && *conf.RPFilter.Enable {
			if conf.RPFilter.Value != nil {
				matched := false
				for _, value := range []int32{0, 1, 2} {
					if *conf.RPFilter.Value == value {
						matched = true
					}
				}
				if !matched {
					conf.RPFilter.Value = pointer.Int32(2)
				}
			} else {
				conf.RPFilter.Value = pointer.Int32(2)
			}
		}
	} else {
		// give default value: RPFilter_Loose(2)
		conf.RPFilter = &ty.RPFilter{
			Enable: pointer.Bool(true),
			Value:  pointer.Int32(2),
		}
	}

	if conf.DelDefaultRoute4 == nil {
		conf.DelDefaultRoute4 = pointer.Bool(true)
	}

	if conf.DelDefaultRoute6 == nil {
		conf.DelDefaultRoute6 = pointer.Bool(true)
	}

	return &conf, nil
}

// delDefaultRoute del default route(ipv4 and ipv6) and add default route to table 100
func movDefaultRoute(netns ns.NetNS, conf *PluginConf) error {
	var err error
	if *conf.DelDefaultRoute4 {
		err = netns.Do(func(_ ns.NetNS) error {
			return updateDefaultRoute(conf.DefaultOverlayInterface, netlink.FAMILY_V4)
		})
	}
	if *conf.DelDefaultRoute6 {
		err = netns.Do(func(_ ns.NetNS) error {
			return updateDefaultRoute(conf.DefaultOverlayInterface, netlink.FAMILY_V6)
		})
	}
	return err
}

// delRoute del default route and add default rule route
// Equivalent: `ip route del <default route>` and `ip r route add <default route> table 100`
func updateDefaultRoute(iface string, ipfamily int) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}
	routes, err := netlink.RouteList(link, ipfamily)
	if err != nil {
		return err
	}

	for _, route := range routes {
		if route.Dst == nil {
			if err = netlink.RouteDel(&route); err != nil {
				return err
			}
			// in order to add route-rule table, we should add rule route table before removing the default route
			// make sure table-100 exist
			route.Table = 100
			if err = netlink.RouteAdd(&route); err != nil {
				return err
			}
		}
	}
	return err
}

func addRoute(netns ns.NetNS, conf *PluginConf) error {
	var err error
	hostIPs, err := utils.GetHostIps()
	if err != nil {
		return err
	}
	err = netns.Do(func(_ ns.NetNS) error {
		// add node ip route
		if err = utils.RouteAdd(conf.DefaultOverlayInterface, hostIPs); err != nil {
			return err
		}
		// add calico/service cidr route
		link, err := netlink.LinkByName(conf.DefaultOverlayInterface)
		if err != nil {
			return err
		}
		for _, route := range conf.Routes {
			if err = netlink.RouteAdd(&netlink.Route{
				LinkIndex: link.Attrs().Index,
				Scope:     netlink.SCOPE_LINK,
				Dst:       &route.Dst,
			}); err != nil {
				return err
			}
		}
		return err
	})
	return err
}

// addRouteRule add route rule for calico cidr(ipv4 and ipv6)
// Equivalent to: `ip rule add ... `
func addRouteRule(netns ns.NetNS, conf *PluginConf) error {
	err := netns.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(conf.DefaultOverlayInterface)
		if err != nil {
			return err
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			if addr.IP.IsMulticast() || addr.IP.IsLinkLocalUnicast() {
				continue
			}
			for _, route := range conf.Routes {
				if utils.IsInSubnet(addr.IP, route.Dst) {
					rule := netlink.NewRule()
					rule.Table = 100
					rule.Src = &route.Dst
					if err = netlink.RuleAdd(rule); err != nil {
						return err
					}
				}
			}

		}
		// we should add rule route table, just like `ip route add default via 169.254.1.1 table 100`
		// but we don't know what's the default route If it has been deleted.
		// so we should add this route rule table before removing the default route
		return err
	})
	return err
}
