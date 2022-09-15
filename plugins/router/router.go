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
	"github.com/spidernet-io/cni-plugins/pkg/constant"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"k8s.io/utils/pointer"
	"os"
	"path/filepath"
)

type PluginConf struct {
	types.NetConf
	// should include: overlay Subnet , clusterIP subnet
	Routes                  []*types.Route `json:"routes,omitempty"`
	HijackOverlayResponse   *bool          `json:"hijack_overlay_response,omitempty"`
	DefaultOverlayInterface string         `json:"overlay_interface,omitempty"`
	// RpFilter
	RPFilter   *ty.RPFilter   `json:"rp_filter,omitempty"`
	Skipped    bool           `json:"skip_call,omitempty"`
	LogOptions *ty.LogOptions `json:"log_options,omitempty"`
}

var binName = filepath.Base(os.Args[0])

var overlayRouteTable = 100

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(binName))
}

func cmdAdd(args *skel.CmdArgs) error {
	var logger *zap.Logger

	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	if err := logging.SetLogOptions(conf.LogOptions); err != nil {
		return fmt.Errorf("faild to init logger: %v ", err)
	}

	logger = logging.LoggerFile.Named(binName)

	k8sArgs := ty.K8sArgs{}
	if err = types.LoadArgs(args.Args, &k8sArgs); nil != err {
		logger.Error(err.Error())
		return fmt.Errorf("failed to get pod information, error=%+v \n", err)
	}

	// register some args into logger
	logger = logger.With(zap.String("Action", "Add"),
		zap.String("ContainerID", args.ContainerID),
		zap.String("PodUID", string(k8sArgs.K8S_POD_UID)),
		zap.String("PodName", string(k8sArgs.K8S_POD_NAME)),
		zap.String("PodNamespace", string(k8sArgs.K8S_POD_NAMESPACE)),
		zap.String("IfName", args.IfName))

	logger.Debug("Succeed to parse cni config", zap.Any("Config", *conf))
	// skip plugin
	if conf.Skipped {
		logger.Info("Ignore this plugin call, Return directly ")
		return types.PrintResult(conf.PrevResult, conf.CNIVersion)
	}
	if conf.PrevResult == nil {
		logger.Error("failed to find PrevResult, must be called as chained plugin")
		return fmt.Errorf("failed to find PrevResult, must be called as chained plugin")
	}

	// ------------------- parse prevResult
	prevResult, err := current.GetResult(conf.PrevResult)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to convert prevResult: %v", err)
	}

	enableIpv4 := false
	enableIpv6 := false
	if len(prevResult.IPs) == 0 {
		err = fmt.Errorf(" got no container IPs")
		logger.Error(err.Error())
		return err
	}
	for _, v := range prevResult.IPs {
		if v.Address.IP.To4() != nil {
			enableIpv4 = true
		} else {
			enableIpv6 = true
		}
	}

	if len(prevResult.Interfaces) == 0 {
		err = fmt.Errorf("failed to find interface from prevResult")
		logger.Error(err.Error())
		return err
	}

	preInterfaceName := prevResult.Interfaces[0].Name
	if len(preInterfaceName) == 0 {
		err = fmt.Errorf("failed to find interface name from prevResult")
		logger.Error(err.Error())
		return err
	}

	// Pass the prevResult through this plugin to the next one
	// result := prevResult

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	// -----------------  Add route table in pod ns
	if enableIpv6 {
		if err := utils.EnableIpv6Sysctl(logger, netns, conf.DefaultOverlayInterface); err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	// hijack overlay response packet to overlay interface
	logger.Debug("Pod IP Family", zap.Bool("ipv4", enableIpv4), zap.Bool("ipv6", enableIpv6))

	if err := hijackResponseRoute(logger, netns, conf, enableIpv4, enableIpv6); err != nil {
		logger.Error(err.Error())
		return err
	}

	// -----------------  Add route table in pod ns
	// add route in pod: hostIP via DefaultOverlayInterface
	if err := addHostIPRoute(logger, netns, conf, enableIpv4, enableIpv6); err != nil {
		logger.Error("failed to add host ip route in container", zap.Error(err))
		return fmt.Errorf("failed to add route: %v", err)
	}

	// add route in pod: custom subnet via DefaultOverlayInterface:  overlay subnet / clusterip subnet ...custom route
	if err := utils.HijackCustomSubnet(logger, netns, conf.Routes, overlayRouteTable, enableIpv4, enableIpv6); err != nil {
		logger.Error(err.Error())
		return err
	}

	// setup sysctl rp_filter
	if err = utils.SysctlRPFilter(logger, netns, conf.RPFilter); err != nil {
		logger.Error(err.Error())
		return err
	}

	// TODO: for multiple macvlan interfaces, maybe need add "ip rule" for second interface

	logger.Info("Succeeded to set for chained interface for overlay interface",
		zap.String("interface", preInterfaceName))
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

	conf.LogOptions = logging.InitLogOptions(conf.LogOptions)
	if conf.LogOptions.LogFilePath == "" {
		conf.LogOptions.LogFilePath = constant.RouterLogDefaultFilePath
	}

	if conf.DefaultOverlayInterface == "" {
		conf.DefaultOverlayInterface = "eth0"
	}

	// value must be 0/1/2
	// If not, giving default value: RPFilter_Loose(2) to it
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
	} else {
		// give default value: RPFilter_Loose(2)
		conf.RPFilter = &ty.RPFilter{
			Enable: pointer.Bool(true),
			Value:  pointer.Int32(2),
		}
	}

	if conf.HijackOverlayResponse == nil {
		conf.HijackOverlayResponse = pointer.Bool(true)
	}

	return &conf, nil
}

// moveRouteTable del default route and add default rule route in pod netns
// Equivalent: `ip route del <default route>` and `ip r route add <default route> table 100`
func moveRouteTable(logger *zap.Logger, iface string, ipfamily int) error {
	logger.Debug(fmt.Sprintf("Moving overlay route table from main table to %d ", overlayRouteTable),
		zap.String("iface", iface),
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

		logger.Debug("Found Route", zap.String("Route", route.String()))
		// only handle route tables from table main
		if route.Table != unix.RT_TABLE_MAIN {
			continue
		}

		// ingore local link route
		if route.Dst.String() == "fe80::/64" {
			continue
		}

		if route.LinkIndex == link.Attrs().Index {
			if route.Dst == nil {
				if err = netlink.RouteDel(&route); err != nil {
					logger.Error("failed to delete default route  in main table ", zap.String("route", route.String()), zap.Error(err))
					return fmt.Errorf("failed to delete default route (%+v) in main table: %+v", route, err)
				}
			}
			route.Table = overlayRouteTable
			if err = netlink.RouteAdd(&route); err != nil {
				logger.Error("failed to add default route to new table ", zap.String("route", route.String()), zap.Error(err))
				return fmt.Errorf("failed to add route (%+v) to new table: %+v", route, err)
			}
			logger.Debug("Succeed to move default route table from main to new table", zap.String("Route", route.String()))
			continue
		} else {
			// especially for ipv6 default gateway
			var generatedRoute, deletedRoute *netlink.Route
			if len(route.MultiPath) == 0 {
				continue
			}

			// get generated default Route for new table
			for _, v := range route.MultiPath {
				logger.Debug("MultiPath", zap.String("v", v.String()), zap.String("Link Index", string(rune(route.LinkIndex))))
				if v.LinkIndex == link.Attrs().Index {
					generatedRoute = &netlink.Route{
						LinkIndex: v.LinkIndex,
						Gw:        v.Gw,
						Table:     overlayRouteTable,
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
			if err = netlink.RouteAdd(generatedRoute); err != nil {
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

// addHostIPRoute add all routes to the node in pod netns, the nexthop is the ip of the host
func addHostIPRoute(logger *zap.Logger, netns ns.NetNS, conf *PluginConf, enableIpv4 bool, enableIpv6 bool) error {
	var err error
	hostIPs, err := utils.GetHostIps(logger, enableIpv4, enableIpv6)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Debug("hijack overlay response packet to overlay interface",
		zap.String("Netns Path", netns.Path()),
		zap.Any("Host IPs", hostIPs),
		zap.Bool("enableIpv4", enableIpv4),
		zap.Bool("enableIpv6", enableIpv6))
	err = netns.Do(func(_ ns.NetNS) error {
		// add route in pod: hostIP via DefaultOverlayInterface
		if _, _, err = utils.RouteAdd(logger, conf.DefaultOverlayInterface, hostIPs, enableIpv4, enableIpv6); err != nil {
			return err
		}
		return nil
	})
	return err
}

// addRouteRule add route rule for calico cidr(ipv4 and ipv6)
// Equivalent to: `ip rule add ... `
func addRouteRule(logger *zap.Logger, netns ns.NetNS, conf *PluginConf, enableIpv4, enableIpv6 bool) error {
	logger.Debug("Add IP Rule Table in Pod Netns", zap.String("Netns Path", netns.Path()))
	err := netns.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(conf.DefaultOverlayInterface)
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			logger.Error(err.Error())
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
			logger.Debug("Added rule table info", zap.String("Rule", rule.String()))
			if err = netlink.RuleAdd(rule); err != nil {
				logger.Error(err.Error())
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

// hijackResponseRoute make sure that the reply packets accessing the overlay interface are still sent from the overlay interface.
func hijackResponseRoute(logger *zap.Logger, netns ns.NetNS, conf *PluginConf, enableIpv4, enableIpv6 bool) error {
	logger.Debug("hijack overlay response packet to overlay interface",
		zap.String("Netns Path", netns.Path()),
		zap.Bool("enableIpv4", enableIpv4),
		zap.Bool("enableIpv6", enableIpv6))
	// add route rule: source overlayIP for new rule
	if err := addRouteRule(logger, netns, conf, enableIpv4, enableIpv6); err != nil {
		logger.Error(fmt.Sprintf("failed to add route table %d: %v ", overlayRouteTable, err))
		return err
	}

	// move overlay default route to table 100
	if *conf.HijackOverlayResponse {
		if enableIpv4 {
			err := netns.Do(func(_ ns.NetNS) error {
				return moveRouteTable(logger, conf.DefaultOverlayInterface, netlink.FAMILY_V4)
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
		}
		if enableIpv6 {
			err := netns.Do(func(_ ns.NetNS) error {
				return moveRouteTable(logger, conf.DefaultOverlayInterface, netlink.FAMILY_V6)
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
		}
	}
	return nil
}
