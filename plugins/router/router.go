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
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"k8s.io/utils/pointer"
	"os"
	"path/filepath"
)

type PluginConf struct {
	types.NetConf
	// should include: overlay Subnet , clusterip subnet
	Routes []*types.Route `json:"routes,omitempty"`
	// RpFilter
	HijackOverlayResponse   *bool  `json:"hijack_overlay_reponse,omitempty"`
	DefaultOverlayInterface string `json:"overlay_interface,omitempty"`
	// RpFilter
	RPFilter *ty.RPFilter `json:"rp_filter,omitempty"`
	Skipped  bool         `json:"skip_call,omitempty"`
}

var binName = filepath.Base(os.Args[0])

var overlayRouteTable = 100

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(binName))
}

var logPrefix string

func cmdAdd(args *skel.CmdArgs) error {

	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	k8sArgs := ty.K8sArgs{}
	if err = types.LoadArgs(args.Args, &k8sArgs); nil != err {
		return fmt.Errorf("failed to get pod information, error=%+v \n", err)
	}
	logPrefix = fmt.Sprintf("[ plugin=%s podNamespace=%s, podName=%s, containerID=%s ]", binName, k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME, args.ContainerID)

	// skip plugin
	if conf.Skipped {
		return types.PrintResult(conf.PrevResult, conf.CNIVersion)
	}
	if conf.PrevResult == nil {
		return fmt.Errorf("%s failed to find PrevResult, must be called as chained plugin", logPrefix)
	}

	// ------------------- parse prevResult
	prevResult, err := current.GetResult(conf.PrevResult)
	if err != nil {
		return fmt.Errorf("%s failed to convert prevResult: %v", logPrefix, err)
	}

	enableIpv4 := false
	enableIpv6 := false
	if len(prevResult.IPs) == 0 {
		return fmt.Errorf("%s got no container IPs", logPrefix)
	} else {
		for _, v := range prevResult.IPs {
			if v.Address.IP.To4() != nil {
				enableIpv4 = true
			} else {
				enableIpv6 = true
			}
		}
	}
	fmt.Fprintf(os.Stderr, "%s enableIpv4=%v, enableIpv6=%v \n", logPrefix, enableIpv4, enableIpv6)

	if len(prevResult.Interfaces) == 0 {
		return fmt.Errorf("%s failed to find interface from prevResult", logPrefix)
	}
	preInterfaceName := prevResult.Interfaces[0].Name
	if len(preInterfaceName) == 0 {
		return fmt.Errorf("%s failed to find interface name from prevResult", logPrefix)
	}

	// Pass the prevResult through this plugin to the next one
	// result := prevResult

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("%s failed to open netns %q: %v", logPrefix, args.Netns, err)
	}
	defer netns.Close()

	// -----------------  Add route table in pod ns
	if enableIpv6 {
		if err := utils.EnableIpv6Sysctl(netns, conf.DefaultOverlayInterface); err != nil {
			return fmt.Errorf("%s failed to enable ipv6 sysctl: %v", logPrefix, err)
		}
	}

	// hijack overlay response packet to overlay interface
	if err := hijackOverlayResponseRoute(netns, conf, enableIpv4, enableIpv6); err != nil {
		return fmt.Errorf("%s failed hijackOverlayResponseRoute: %v", logPrefix, err)
	}
	fmt.Fprintf(os.Stderr, "%s succeeded to hijack Overlay Response Route \n", logPrefix)

	// add route in pod: hostIP via DefaultOverlayInterface
	// add route in pod: custom subnet via DefaultOverlayInterface:  overlay subnet / clusterip subnet ...custom route
	if err := addRoute(netns, conf, enableIpv4, enableIpv6); err != nil {
		return fmt.Errorf("%s failed to add route: %v", logPrefix, err)
	}

	fmt.Fprintf(os.Stderr, "%s succeeded to add route for chained interface %s \n", logPrefix, preInterfaceName)

	// 4. setup sysctl rp_filter
	if err = utils.SysctlRPFilter(netns, conf.RPFilter); err != nil {
		return fmt.Errorf("%s failed to set rp_filter : %v", logPrefix, err)
	}
	fmt.Fprintf(os.Stderr, "%s succeeded to set rp_filter \n", logPrefix)

	// TODO: for multiple macvlan interfaces, maybe need add "ip rule" for second interface

	fmt.Fprintf(os.Stderr, "%s succeeded to set for chained interface %s for overlay interface %s \n", logPrefix, preInterfaceName, conf.DefaultOverlayInterface)

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
	if conf.RPFilter == nil {
		// give default value: RPFilter_Loose(2)
		conf.RPFilter = &ty.RPFilter{
			Enable: pointer.Bool(true),
			Value:  pointer.Int32(2),
		}
	}
	if conf.RPFilter != nil {
		if conf.RPFilter.Enable == nil {
			// give default value: RPFilter_Loose(2)
			conf.RPFilter = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			}
		}
		if conf.RPFilter.Enable != nil {
			if *conf.RPFilter.Enable {
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
		}
	}

	if conf.HijackOverlayResponse == nil {
		conf.HijackOverlayResponse = pointer.Bool(true)
	}

	return &conf, nil
}

// delRoute del default route and add default rule route
// Equivalent: `ip route del <default route>` and `ip r route add <default route> table 100`
func moveOverlayRoute(iface string, ipfamily int) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}
	routes, err := netlink.RouteList(nil, ipfamily)
	if err != nil {
		return err
	}

	for _, route := range routes {
		fmt.Fprintf(os.Stderr, "%s [welan] route: %+v \n", logPrefix, route)

		// in order to add route-rule table, we should add rule route table before removing the default route
		// make sure table-100 exist
		if route.Table != unix.RT_TABLE_MAIN {
			continue
		}

		if route.LinkIndex != link.Attrs().Index {
			// especially for ipv6 default gateway
			var generatedRoute, modifiedMainDefaultRoute *netlink.Route
			if len(route.MultiPath) == 0 {
				continue
			}
			fmt.Fprintf(os.Stderr, "%s [welan 2 ] route MultiPath : %+v , link.Attrs().Index =%v \n", logPrefix, route.MultiPath, link.Attrs().Index)

			// get generated default Route for new table
			for _, v := range route.MultiPath {
				fmt.Fprintf(os.Stderr, "%s [welan 2.5 ] route route : %+v  \n", logPrefix, v)

				if v.LinkIndex == link.Attrs().Index {
					generatedRoute = &netlink.Route{
						LinkIndex: route.LinkIndex,
						Gw:        v.Gw,
						Table:     overlayRouteTable,
						MTU:       route.MTU,
					}
					break
				}
			}
			if generatedRoute == nil {
				continue
			}

			// get generated default Route for main table
			for _, v := range route.MultiPath {
				if v.LinkIndex != link.Attrs().Index {
					modifiedMainDefaultRoute = &netlink.Route{
						LinkIndex: route.LinkIndex,
						Gw:        v.Gw,
						Table:     unix.RT_TABLE_MAIN,
						MTU:       route.MTU,
					}
					break
				}
			}

			fmt.Fprintf(os.Stderr, "%s [welan 3 ] route generatedRoute : %+v \n", logPrefix, generatedRoute)
			fmt.Fprintf(os.Stderr, "%s [welan 3 ] route modifiedMainDefaultRoute : %+v \n", logPrefix, modifiedMainDefaultRoute)
			fmt.Fprintf(os.Stderr, "%s [welan 3 ] route delete route : %+v \n", logPrefix, route)

			// add to new table
			if err = netlink.RouteAdd(generatedRoute); err != nil {
				return err
			}
			// delete original default
			if err = netlink.RouteDel(&route); err != nil {
				return err
			}
			// set new default for main
			if err = netlink.RouteAdd(modifiedMainDefaultRoute); err != nil {
				return err
			}
		} else {
			// clean default route in main table but keep 169.254.1.1
			if route.Dst == nil {
				if err = netlink.RouteDel(&route); err != nil {
					return err
				}
			}
			route.Table = overlayRouteTable
			if err = netlink.RouteAdd(&route); err != nil {
				return err
			}
		}
	}
	return err
}

func addRoute(netns ns.NetNS, conf *PluginConf, enableIpv4 bool, enableIpv6 bool) error {
	var err error

	hostIPs, err := utils.GetHostIps()
	if err != nil {
		return err
	}

	err = netns.Do(func(_ ns.NetNS) error {

		// add route in pod: hostIP via DefaultOverlayInterface
		if err = utils.RouteAdd(conf.DefaultOverlayInterface, hostIPs, enableIpv4, enableIpv6); err != nil {
			return err
		}

		// add route in pod: custom subnet via DefaultOverlayInterface:  overlay subnet / clusterip subnet ...custom route
		link, err := netlink.LinkByName(conf.DefaultOverlayInterface)
		if err != nil {
			return err
		}

		for _, route := range conf.Routes {
			if err = netlink.RouteAdd(&netlink.Route{
				LinkIndex: link.Attrs().Index,
				Scope:     netlink.SCOPE_LINK,
				Dst:       &route.Dst,
			}); err != nil && err.Error() != "file exists" {
				return err
			}
		}
		return err
	})
	return err
}

// addRouteRule add route rule for calico cidr(ipv4 and ipv6)
// Equivalent to: `ip rule add ... `
func addRouteRule(netns ns.NetNS, conf *PluginConf, enableIpv4, enableIpv6 bool) error {
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
			if addr.IP.To4() != nil && !enableIpv4 {
				continue
			}
			if addr.IP.To16() != nil && !enableIpv6 {
				continue
			}
			rule := netlink.NewRule()
			rule.Table = overlayRouteTable
			rule.Src = addr.IPNet
			if err = netlink.RuleAdd(rule); err != nil {
				return err
			}
		}
		// we should add rule route table, just like `ip route add default via 169.254.1.1 table 100`
		// but we don't know what's the default route If it has been deleted.
		// so we should add this route rule table before removing the default route
		return err
	})
	return err
}

func hijackOverlayResponseRoute(netns ns.NetNS, conf *PluginConf, enableIpv4, enableIpv6 bool) error {

	// set route rule: source overlayIP for new rule
	if err := addRouteRule(netns, conf, enableIpv4, enableIpv6); err != nil {
		return err
	}

	// move overlay default route to table 100
	if *conf.HijackOverlayResponse {
		if enableIpv4 {
			err := netns.Do(func(_ ns.NetNS) error {
				return moveOverlayRoute(conf.DefaultOverlayInterface, netlink.FAMILY_V4)
			})
			if err != nil {
				return err
			}
		}
		if enableIpv6 {
			err := netns.Do(func(_ ns.NetNS) error {
				return moveOverlayRoute(conf.DefaultOverlayInterface, netlink.FAMILY_V6)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
