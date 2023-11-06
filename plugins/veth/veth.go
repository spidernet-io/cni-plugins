// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spidernet-io/cni-plugins/pkg/config"
	"github.com/spidernet-io/cni-plugins/pkg/constant"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/spidernet-io/cni-plugins/pkg/networking"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/cni-plugins/pkg/utils"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	spiderpool "github.com/spidernet-io/spiderpool/pkg/networking/networking"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

type PluginConf struct {
	types.NetConf
	// should include: overlay Subnet , clusterip subnet
	OverlayHijackSubnet    []string `json:"overlay_hijack_subnet,omitempty"`
	ServiceHijackSubnet    []string `json:"service_hijack_subnet,omitempty"`
	AdditionalHijackSubnet []string `json:"additional_hijack_subnet,omitempty"`
	// RpFilter
	RPFilter     *ty.RPFilter     `json:"rp_filter,omitempty" `
	Skipped      bool             `json:"skip_call,omitempty"`
	MigrateRoute *ty.MigrateRoute `json:"migrate_route,omitempty"`
	LogOptions   *ty.LogOptions   `json:"log_options,omitempty"`
	IPConflict   *ty.IPConflict   `json:"ip_conflict,omitempty"`
	MacPrefix    string           `json:"mac_prefix,omitempty"`
	OnlyOpMac    bool             `json:"only_op_mac,omitempty"`
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
	startTime := time.Now()

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

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	logger.Debug("Get prevResult", zap.Any("prevResult", prevResult))

	// we do check if ip is conflict firstly
	if conf.IPConflict != nil && conf.IPConflict.Enabled {
		err = networking.DoIPConflictChecking(logger, netns, args.IfName, prevResult.IPs, conf.IPConflict)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	if len(conf.MacPrefix) != 0 {
		newMac, err := utils.OverwriteMacAddress(logger, netns, conf.MacPrefix, args.IfName)
		if err != nil {
			return fmt.Errorf("failed to update mac address, maybe mac_prefix is invalid: %v", conf.MacPrefix)
		}
		logger.Info("Update mac address successfully", zap.String("interface", constant.DefaultInterfaceName), zap.String("new mac", newMac))
		if conf.OnlyOpMac {
			logger.Debug("only update mac address, exiting now...")
			return types.PrintResult(conf.PrevResult, conf.CNIVersion)
		}
	}

	enableIpv4, enableIpv6 := false, false
	ipfamily := -1
	for _, v := range prevResult.IPs {
		if v.Address.IP.To4() != nil {
			enableIpv4 = true
			ipfamily = netlink.FAMILY_V4
		} else {
			enableIpv6 = true
			ipfamily = netlink.FAMILY_V6
		}
	}

	if enableIpv4 && enableIpv6 {
		ipfamily = netlink.FAMILY_ALL
	}

	if len(prevResult.Interfaces) == 0 {
		err = fmt.Errorf("failed to find interface from prevResult")
		logger.Error(err.Error())
		return err
	}
	chainedInterface := prevResult.Interfaces[0].Name
	if len(chainedInterface) == 0 {
		err = fmt.Errorf("failed to find interface name from prevResult")
		logger.Error(err.Error())
		return err
	}

	// Pass the prevResult through this plugin to the next one
	// result := prevResult

	isfirstInterface, e := utils.CheckInterfaceMiss(netns, defaultConVeth)
	if e != nil {
		logger.Error("failed to check first veth interface", zap.Error(e))
		return fmt.Errorf("failed to check first veth interface: %v", e)
	}

	if !isfirstInterface {
		logger.Info("Start call veth as the addon plugin", zap.Any("config", conf))
	} else {
		logger.Info("Start call veth as first plugin", zap.Any("config", conf))
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

	// get all ip of pod
	var allPodIp []netlink.Addr
	err = netns.Do(func(netNS ns.NetNS) error {
		allPodIp, err = spiderpool.GetAllIPAddress(ipfamily, []string{`^lo$`})
		if err != nil {
			logger.Error("failed to GetAllIPAddress in pod", zap.Error(err))
			return fmt.Errorf("failed to GetAllIPAddress in pod: %v", err)
		}
		return nil
	})
	if err != nil {
		logger.Error("failed to all ip of pod", zap.Error(err))
		return err
	}
	logger.Info("Succeed to get ips from given interface inside container", zap.String("interface", chainedInterface), zap.Any("container ips", allPodIp))

	// get ip addresses of the node
	hostIPs, err := networking.GetAllHostIPRouteForPod(ipfamily, allPodIp)
	if err != nil {
		logger.Error("failed to get IPAddressOnNode", zap.Error(err))
		return fmt.Errorf("failed to get IPAddressOnNode: %v", err)
	}
	logger.Debug("success get host IP for route to Pod", zap.Any("hostIPs", hostIPs))

	if enableIpv6 {
		if err := utils.EnableIpv6Sysctl(logger, netns); err != nil {
			return err
		}
	}

	currentIPs, err := spiderpool.IPAddressByName(netns, args.IfName, ipfamily)
	if err != nil {
		logger.Error(err.Error())
		return fmt.Errorf("failed to IPAddressByName for pod %s : %v", args.IfName, err)
	}

	// 2. setup neighborhood
	if err = setupNeighborhood(logger, isfirstInterface, netns, chainedInterface, hostInterface, conInterface, hostIPs, currentIPs, args.ContainerID); err != nil {
		logger.Error(err.Error())
		return err
	}

	ruleTable := unix.RT_TABLE_MAIN
	if !isfirstInterface {
		ruleTable = utils.GetRuleNumber(chainedInterface)
		if ruleTable < 0 {
			logger.Error("failed to get the number of rule table for interface", zap.String("interface", chainedInterface))
			return fmt.Errorf("failed to get the number of rule table for interface %s", chainedInterface)
		}
	}

	// 3. setup routes
	if err = setupRoutes(logger, netns, ruleTable, ipfamily, hostInterface, conInterface, hostIPs, currentIPs, conf); err != nil {
		logger.Error(err.Error())
		return err
	}

	//4. migrate default route
	if !isfirstInterface {
		if err = utils.MigrateRoute(logger, netns, chainedInterface, chainedInterface, currentIPs, *conf.MigrateRoute, ruleTable, enableIpv4, enableIpv6); err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	// 5. setup sysctl rp_filter
	if err = utils.SysctlRPFilter(logger, netns, conf.RPFilter); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("succeeded to call veth-plugin", zap.Int64("Time Cost", time.Since(startTime).Microseconds()))
	return types.PrintResult(conf.PrevResult, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
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

	logger.Debug("Start call veth cmdDel", zap.Any("config", conf))

	hostVeth := getHostVethName(args.ContainerID)
	vethLink, err := netlink.LinkByName(hostVeth)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			logger.Debug("Host veth has gone, nothing to do", zap.String("HostVeth", hostVeth))
			return nil
		}
		return fmt.Errorf("failed to get host veth device %s: %w", hostVeth, err)
	}

	if err = netlink.LinkDel(vethLink); err != nil {
		logger.Error("failed to del hostVeth", zap.Error(err))
		return fmt.Errorf("failed to del hostVeth %s: %w", hostVeth, err)
	}

	logger.Debug("Success to call veth cmdDel", zap.Any("config", conf))
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	// TODO
	return fmt.Errorf("not implement it")
}

// parseConfig parses the supplied configuration (and prevResult) from stdin.
func parseConfig(stdin []byte) (*PluginConf, error) {
	var err error
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
	if err = config.ValidateOverwriteMacAddress(conf.MacPrefix); err != nil {
		return nil, err
	}

	if conf.IPConflict != nil {
		conf.IPConflict = config.ValidateIPConflict(conf.IPConflict)
		_, err = time.ParseDuration(conf.IPConflict.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval %s: %v, input like: 1s or 1m", conf.IPConflict.Interval, err)
		}
	}

	conf.LogOptions = logging.InitLogOptions(conf.LogOptions)
	if conf.LogOptions.LogFilePath == "" {
		conf.LogOptions.LogFilePath = constant.VethLogDefaultFilePath
	}

	if conf.OnlyOpMac {
		return &conf, nil
	}

	conf.ServiceHijackSubnet, conf.OverlayHijackSubnet, err = config.ValidateRoutes(conf.ServiceHijackSubnet, conf.OverlayHijackSubnet)
	if err != nil {
		return nil, err
	}

	// value must be 0/1/2
	// If not, giving default value: RPFilter_Loose(2) to it
	conf.RPFilter = config.ValidateRPFilterConfig(conf.RPFilter)

	conf.MigrateRoute = config.ValidateMigrateRouteConfig(conf.MigrateRoute)

	return &conf, nil
}

// setupVeth sets up a pair of virtual ethernet devices. It will create both veth
// devices and move the host-side veth into the provided hostNS namespace.
func setupVeth(logger *zap.Logger, netns ns.NetNS, isfirstInterface bool, containerID string, pr *current.Result) (*current.Interface, *current.Interface, error) {
	hostInterface := &current.Interface{Name: getHostVethName(containerID)}
	containerInterface := &current.Interface{}
	err := netns.Do(func(hostNS ns.NetNS) error {
		if !isfirstInterface {
			link, err := netlink.LinkByName(defaultConVeth)
			if err != nil {
				return err
			}
			containerInterface.Mac = link.Attrs().HardwareAddr.String()
			containerInterface.Name = defaultConVeth
			logger.Info("Veth-peer has already setup, skip setupVeth ")
			return nil
		}
		hostVeth, contVeth0, err := ip.SetupVethWithName(defaultConVeth, hostInterface.Name, defaultMtu, "", hostNS)
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
func setupNeighborhood(logger *zap.Logger, isfirstInterface bool, netns ns.NetNS, chainInterface string, hostInterface, chainedInterface *current.Interface, hostIPs []net.IP, conIPs []netlink.Addr, containerId string) error {
	var err error
	// set neighborhood on host
	logger.Debug("Add Neighborhood Table In Host Side",
		zap.String("hostInterface name", hostInterface.Name),
		zap.String("containerInterface Mac", chainedInterface.Mac),
		zap.Any("container IPs", conIPs))

	hostVethLink, err := netlink.LinkByName(hostInterface.Name)
	if err != nil {
		logger.Error(fmt.Sprintf("setupNeighborhood: %v", err))
		return fmt.Errorf("setupNeighborhood: %v", err)
	}
	hostInterface.Mac = hostVethLink.Attrs().HardwareAddr.String()

	for _, conIP := range conIPs {
		hw, err := net.ParseMAC(chainedInterface.Mac)
		if err != nil {
			return fmt.Errorf("veth's mac is invalid: %v", err)
		}

		if err = spiderpool.AddStaticNeighborTable(hostVethLink.Attrs().Index, conIP.IP, hw); err != nil {
			logger.Error(err.Error())
			return err
		}
	}
	if !isfirstInterface {
		logger.Debug("Succeed to add neighbor table for interface", zap.String("chainInterface", chainInterface), zap.Any("Container interface ips", conIPs))
		return nil
	}

	err = netns.Do(func(_ ns.NetNS) error {
		logger.Debug("Add HostpIPs Neighborhood Table In Pod Side",
			zap.String("defaultConVeth", defaultConVeth),
			zap.String("hostInterface veth Mac", hostInterface.Mac),
			zap.String("hostVethLink Mac", hostVethLink.Attrs().HardwareAddr.String()))

		podVethLink, err := netlink.LinkByName(defaultConVeth)
		if err != nil {
			logger.Error(fmt.Sprintf("setupNeighborhood: %v", err))
			return fmt.Errorf("setupNeighborhood: %v", err)
		}

		for _, hostIP := range hostIPs {
			if err = spiderpool.AddStaticNeighborTable(podVethLink.Attrs().Index, hostIP, hostVethLink.Attrs().HardwareAddr); err != nil {
				logger.Error(err.Error())
				return err
			}
		}
		return nil
	})

	return err
}

// setupRoutes setup routes for pod and host
// equivalent to: `ip route add $route`
func setupRoutes(logger *zap.Logger, netns ns.NetNS, ruleTable, ipfamily int, hostInterface, chainedInterface *current.Interface, hostIPs []net.IP, conIPs []netlink.Addr, conf *PluginConf) error {
	v4Gw, v6Gw, err := spiderpool.GetGatewayIP(conIPs)
	if err != nil {
		logger.Error("failed to GetGatewayIP", zap.Error(err))
		return err
	}

	// set routes for pod
	err = netns.Do(func(_ ns.NetNS) error {
		// add host ip route
		// equiva to "ip r add hostIP dev veth0 table <ruleTable> "
		for _, hostAddress := range hostIPs {
			ipNet := spiderpool.ConvertMaxMaskIPNet(hostAddress)
			if err = spiderpool.AddRoute(logger, ruleTable, ipfamily, netlink.SCOPE_LINK, defaultConVeth, ipNet, nil, nil); err != nil {
				logger.Error("failed to AddRoute for ipAddressOnNode", zap.Error(err))
				return fmt.Errorf("failed to AddRouteTable for ipAddressOnNode: %v", err)
			}
		}

		allSubnets := append(conf.ServiceHijackSubnet, conf.OverlayHijackSubnet...)
		allSubnets = append(allSubnets, conf.AdditionalHijackSubnet...)
		for _, hijack := range allSubnets {
			nip, ipNet, err := net.ParseCIDR(hijack)
			if err != nil {
				logger.Error("Invalid Hijack Cidr", zap.String("Cidr", hijack), zap.Error(err))
				return err
			}

			if nip.To4() != nil && v4Gw == nil {
				logger.Warn("ignore adding hijack routing table(ipv4), due to ipv4 gateway is nil", zap.String("IPv4 Hijack cidr", hijack))
				continue
			}

			if nip.To4() == nil && v6Gw == nil {
				logger.Warn("ignore adding hijack routing table(ipv6), due to ipv6 gateway is nil", zap.String("IPv6 Hijack cidr", hijack))
				continue
			}

			if err := spiderpool.AddRoute(logger, ruleTable, ipfamily, netlink.SCOPE_UNIVERSE, defaultConVeth, ipNet, v4Gw, v6Gw); err != nil {
				logger.Error("failed to AddRoute for hijackCIDR", zap.String("Dst", ipNet.String()), zap.Error(err))
				return fmt.Errorf("failed to AddRoute for hijackCIDR: %v", err)
			}

		}
		logger.Debug("AddRouteTable for localCIDRs successfully", zap.Strings("localCIDRs", allSubnets))
		return nil
	})

	for idx := range conIPs {
		ipNet := spiderpool.ConvertMaxMaskIPNet(conIPs[idx].IP)

		// set routes for host
		// equivalent: ip add  <chainedIPs> dev <hostVethName> table  on host
		if err = spiderpool.AddRoute(logger, unix.RT_TABLE_MAIN, ipfamily, netlink.SCOPE_LINK, hostInterface.Name, ipNet, nil, nil); err != nil {
			logger.Error("failed to AddRouteTable for preInterface IPAddress", zap.Error(err))
			return fmt.Errorf("failed to AddRouteTable for preInterface %s's IPAddress: %v", hostInterface.Name, err)
		}
		logger.Info("add route for to pod in host", zap.String("Dst", ipNet.String()))
	}

	return err
}
