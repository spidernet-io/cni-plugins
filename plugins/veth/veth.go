// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"github.com/spidernet-io/cni-plugins/pkg/constant"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"k8s.io/utils/pointer"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

type PluginConf struct {
	types.NetConf
	// should include: overlay Subnet , clusterip subnet
	Routes *ty.Route `json:"routes,omitempty"`
	// RpFilter
	RPFilter   *ty.RPFilter   `json:"rp_filter,omitempty"`
	Skipped    bool           `json:"skip_call,omitempty"`
	LogOptions *ty.LogOptions `json:"log_options,omitempty"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

var binName = filepath.Base(os.Args[0])

// the interface added by this plugin
var defaultConVeth = "veth0"

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

	k8sArgs := ty.K8sArgs{}
	if err = types.LoadArgs(args.Args, &k8sArgs); nil != err {
		return fmt.Errorf("failed to get pod information, error=%+v \n", err)
	}

	logger = logging.LoggerFile.Named(binName)

	// register some args into logger
	logger = logger.With(zap.String("Action", "Add"),
		zap.String("ContainerID", args.ContainerID),
		zap.String("PodUID", string(k8sArgs.K8S_POD_UID)),
		zap.String("PodName", string(k8sArgs.K8S_POD_NAME)),
		zap.String("PodNamespace", string(k8sArgs.K8S_POD_NAMESPACE)),
		zap.String("IfName", args.IfName))

	// skip veth plugin
	if conf.Skipped {
		logger.Info("Ignore this plugin call, Return directly ")
		return types.PrintResult(conf.PrevResult, conf.CNIVersion)
	}
	if conf.PrevResult == nil {
		logger.Error("failed to find PrevResult, must be called as chained plugin")
		return fmt.Errorf("failed to find PrevResult, must be called as chained plugin")
	}

	prevResult, err := current.GetResult(conf.PrevResult)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to convert prevResult: %v", err)
	}

	logger.Debug("Start call veth", zap.Any("config", conf))

	if len(prevResult.IPs) == 0 {
		err = fmt.Errorf(" got no container IPs")
		logger.Error(err.Error())
		return err
	}

	enableIpv4, enableIpv6 := false, false
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

	logger.Debug("Pod IP Family", zap.Bool("ipv4", enableIpv4), zap.Bool("ipv6", enableIpv6))

	// Pass the prevResult through this plugin to the next one
	// result := prevResult

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	// Check if the veth-plugin has been executed
	// if so, skip it
	isfirstInterface, e := utils.CheckInterfaceMiss(netns, defaultConVeth)
	if e != nil {
		logger.Error("failed to check first veth interface", zap.Error(e))
		return fmt.Errorf("failed to check first veth interface: %v", e)
	}

	// 1. setup veth pair
	var hostInterface *current.Interface
	var conInterface *current.Interface
	hostInterface, conInterface, err = setupVeth(logger, netns, isfirstInterface, args.ContainerID, prevResult)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	logger.Info("Succeeded to set veth interface", zap.Any("interfaces", prevResult.Interfaces), zap.Any("ips", prevResult.IPs), zap.Any("routes", prevResult.Routes))

	// get all ips on the node
	hostIPs, err := utils.GetHostIps(logger, enableIpv4, enableIpv6)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to get host ips: %v", err)
	}

	// get ips from pod
	conIPs, err := getChainedInterfaceIps(netns, preInterfaceName)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to find ip from chained interface %s : %v", preInterfaceName, err)
	}
	logger.Info("Succeed to get ips from given interface inside container", zap.String("interface", preInterfaceName), zap.Strings("container ips", conIPs))

	if enableIpv6 {
		if err := utils.EnableIpv6Sysctl(logger, netns, defaultConVeth); err != nil {
			return err
		}
	}

	// 2. setup neighborhood
	if err = setupNeighborhood(logger, isfirstInterface, netns, hostInterface, conInterface, hostIPs, conIPs, args.ContainerID); err != nil {
		logger.Error(err.Error())
		return err
	}

	// 3. setup routes
	if err = setupRoutes(logger, netns, hostInterface, conInterface, hostIPs, conIPs, conf.Routes, enableIpv4, enableIpv6); err != nil {
		logger.Error(err.Error())
		return err
	}

	// 4. setup sysctl rp_filter
	if err = utils.SysctlRPFilter(logger, netns, conf.RPFilter); err != nil {
		logger.Error(err.Error())
		return err
	}

	// TODO: for multiple macvlan interfaces, maybe need add "ip rule" for second interface
	logger.Info("succeeded to call veth-plugin")
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
		return nil, fmt.Errorf("[veth] failed to parse network configuration: %v", err)
	}

	// Parse previous result. This will parse, validate, and place the
	// previous result object into conf.PrevResult. If you need to modify
	// or inspect the PrevResult you will need to convert it to a concrete
	// versioned Result struct.
	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("[veth] could not parse prevResult: %v", err)
	}
	// End previous result parsing

	// some validation
	if len(conf.Routes.OverlaySubnet) == 0 {
		return nil, fmt.Errorf("the subnet of overlay cni(such as calico or cilium) must be given")
	}

	if len(conf.Routes.ServiceSubnet) == 0 {
		return nil, fmt.Errorf("the subnet of service clusterip must be given")
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
	conf.LogOptions = logging.InitLogOptions(conf.LogOptions)
	if conf.LogOptions.LogFilePath == "" {
		conf.LogOptions.LogFilePath = constant.VethLogDefaultFilePath
	}
	return &conf, nil
}

// setupVeth sets up a pair of virtual ethernet devices. It will create both veth
// devices and move the host-side veth into the provided hostNS namespace.
func setupVeth(logger *zap.Logger, netns ns.NetNS, isfirstInterface bool, containerID string, pr *current.Result) (*current.Interface, *current.Interface, error) {
	if !isfirstInterface {
		logger.Info("Veth-peer has already setup, skip it ")
		return nil, nil, nil
	}

	hostInterface := &current.Interface{}
	containerInterface := &current.Interface{}

	hostVethName := getHostVethName(containerID)
	err := netns.Do(func(hostNS ns.NetNS) error {
		hostVeth, contVeth0, err := ip.SetupVethWithName(defaultConVeth, hostVethName, defaultMtu, "", hostNS)
		if err != nil {
			return fmt.Errorf("[veth] failed to set veth peer: %v", err)
		}
		hostInterface.Name = hostVeth.Name
		hostInterface.Mac = hostVeth.HardwareAddr.String()
		containerInterface.Name = contVeth0.Name
		containerInterface.Mac = contVeth0.HardwareAddr.String()
		containerInterface.Sandbox = netns.Path()

		pr.Interfaces = append(pr.Interfaces, hostInterface, containerInterface)

		if err = setLinkup(contVeth0.Name); err != nil {
			return fmt.Errorf("[veth] failed to set %s up: %v", contVeth0.Name, err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return hostInterface, containerInterface, nil
}

// setupNeighborhood setup neighborhood tables for pod and host.
// equivalent to: `ip neigh add ....`
func setupNeighborhood(logger *zap.Logger, isfirstInterface bool, netns ns.NetNS, hostInterface, chainedInterface *current.Interface, hostIPs, conIPs []string, containerId string) error {
	var err error
	// set neighborhood on host
	logger.Debug("Add Neighborhood Table In Host Side", zap.String("Netns Path", netns.Path()),
		zap.String("hostInterface name", hostInterface.Name),
		zap.String("containerInterface Mac", chainedInterface.Mac),
		zap.Strings("container IPs", conIPs))

	//if !isfirstInterface {
	//	//
	//	logger.Debug("")
	//}

	if err = neighborAdd(logger, hostInterface.Name, chainedInterface.Mac, conIPs); err != nil {
		logger.Error(err.Error())
		return err
	}

	// set up host veth Interface neighborhood in pod
	// bug?: sometimes hostveth's mac not be correct, we get hosVeth's mac via LinkByName directly
	hostVethLink, err := netlink.LinkByName(getHostVethName(containerId))
	if err != nil {
		logger.Error("failed to find veth peer host side", zap.String("Veth Name(host)", getHostVethName(containerId)), zap.Error(err))
		return err
	}
	err = netns.Do(func(_ ns.NetNS) error {
		logger.Debug("Add Neighborhood Table In Pod Side", zap.String("Netns Path", netns.Path()),
			zap.String("Container defaultConVeth", defaultConVeth),
			zap.String("hostInterface veth Mac", hostInterface.Mac),
			zap.String("hostVethLink Mac", hostVethLink.Attrs().HardwareAddr.String()),
			zap.Strings("Host IPs", hostIPs))
		if err := neighborAdd(logger, defaultConVeth, hostVethLink.Attrs().HardwareAddr.String(), hostIPs); err != nil {
			logger.Error(err.Error())
			return err
		}
		return nil
	})

	return err
}

// setupRoutes setup routes for pod and host
// equivalent to: `ip route add $route`
func setupRoutes(logger *zap.Logger, netns ns.NetNS, hostInterface, chainedInterface *current.Interface, hostIPs, conIPs []string, routes *ty.Route, enableIpv4, enableIpv6 bool) error {
	var err error
	// set routes for host
	if _, _, err = utils.RouteAdd(logger, hostInterface.Name, conIPs, enableIpv4, enableIpv6); err != nil {
		logger.Error("[host side]", zap.Error(err))
		return fmt.Errorf("[host side]: %v", err)
	}

	// set routes for pod
	err = netns.Do(func(_ ns.NetNS) error {
		// add host ip route
		// equiva to "ip r add hostIP dev veth"
		if _, _, err = utils.RouteAdd(logger, chainedInterface.Name, hostIPs, enableIpv4, enableIpv6); err != nil {
			logger.Error("[pod side]", zap.Error(err))
			return fmt.Errorf("[pod side]: %v", err)
		}

		// setup custom routes from cni conf
		// such as calico cidr, service cidr
		// ip route add <route> via hostIP dev veth0
		link, err := netlink.LinkByName(chainedInterface.Name)
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		// get one host ipv4 ip and one host ipv6 ip, as destination (if exist)
		ipv4, ipv6 := false, false
		viaIPs := make([]string, 0, 2)
		for _, ip := range hostIPs {
			netIP := net.ParseIP(ip)
			ipv4, ipv6, viaIPs = filterIPs(netIP, ipv4, ipv6, viaIPs)
		}

		var destHostIpv4, destHostIpv6 *net.IP
		for _, viaIP := range viaIPs {
			netIP := net.ParseIP(viaIP)
			if netIP.To4() != nil {
				destHostIpv4 = &netIP
			} else {
				destHostIpv6 = &netIP
			}
		}
		if err = addSubnetRoute(logger, routes.ServiceSubnet, link.Attrs().Index, enableIpv4, enableIpv6, destHostIpv4, destHostIpv6); err != nil {
			return err
		}
		logger.Debug("Succeed to add service subnet to pod side", zap.Strings("Service Subnet", routes.ServiceSubnet))
		if err = addSubnetRoute(logger, routes.OverlaySubnet, link.Attrs().Index, enableIpv4, enableIpv6, destHostIpv4, destHostIpv6); err != nil {
			return err
		}
		logger.Debug("Succeed to add overlay subnet to pod side", zap.Strings("Overlay CNI Subnet", routes.ServiceSubnet))

		if err = addSubnetRoute(logger, routes.CustomSubnet, link.Attrs().Index, enableIpv4, enableIpv6, destHostIpv4, destHostIpv6); err != nil {
			return err
		}
		return nil
	})
	return err
}

func addSubnetRoute(logger *zap.Logger, routes []string, linkIndex int, enableIpv4, enableIpv6 bool, destHostIpv4, destHostIpv6 *net.IP) error {
	for _, route := range routes {
		_, ipNet, err := net.ParseCIDR(route)
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		gw := net.IP{}
		if ipNet.IP.To4() != nil && enableIpv4 && destHostIpv4 != nil {
			gw = *destHostIpv4
		}
		if ipNet.IP.To4() == nil && enableIpv6 && destHostIpv6 != nil {
			gw = *destHostIpv6
		}
		if len(gw) == 0 {
			logger.Warn("the route given does not match the ipversion of the pod, ignore the creation of this route", zap.String("route", ipNet.String()))
			continue
		}
		if err = netlink.RouteAdd(&netlink.Route{
			LinkIndex: linkIndex,
			Dst:       ipNet,
			Gw:        gw,
		}); err != nil {
			logger.Error("[pod side]failed to add route", zap.String("dst", ipNet.String()), zap.String("gw", gw.String()), zap.Error(err))
			return fmt.Errorf("[pod side] failed to add route: %v: %v ", ipNet.String(), err)
		}
	}
	return nil
}
