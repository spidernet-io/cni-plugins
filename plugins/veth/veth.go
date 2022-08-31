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
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
	"github.com/vishvananda/netlink"
	"k8s.io/utils/pointer"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

type PluginConf struct {
	types.NetConf
	// should include: overlay Subnet , clusterip subnet
	Routes []*types.Route `json:"routes,omitempty"`
	// RpFilter
	RPFilter *ty.RPFilter `json:"rp_filter,omitempty"`
	Skipped  bool         `json:"skip_call,omitempty"`
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
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	k8sArgs := ty.K8sArgs{}
	if err = types.LoadArgs(args.Args, &k8sArgs); nil != err {
		return fmt.Errorf("failed to get pod information, error=%+v \n", err)
	}
	logPrefix := fmt.Sprintf("[ plugin=%s podNamespace=%s, podName=%s, containerID=%s ]", binName, k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME, args.ContainerID)

	// skip veth plugin
	if conf.Skipped {
		return types.PrintResult(conf.PrevResult, conf.CNIVersion)
	}
	if conf.PrevResult == nil {
		return fmt.Errorf("%s failed to find PrevResult, must be called as chained plugin", logPrefix)
	}

	prevResult, err := current.GetResult(conf.PrevResult)
	if err != nil {
		return fmt.Errorf("%s failed to convert prevResult: %v", logPrefix, err)
	}

	if len(prevResult.IPs) == 0 {
		return fmt.Errorf("%s got no container IPs", logPrefix)
	}

	if len(prevResult.Interfaces) == 0 {
		return fmt.Errorf("%s failed to find interface from prevResult", logPrefix)
	}
	preInterfaceName := prevResult.Interfaces[0].Name
	if len(preInterfaceName) == 0 {
		return fmt.Errorf("%s failed to find interface name from prevResult", logPrefix)
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

	// Pass the prevResult through this plugin to the next one
	// result := prevResult

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("%s failed to open netns %q: %v", logPrefix, args.Netns, err)
	}
	defer netns.Close()

	// Check if the veth-plugin has been executed
	// if so, skip it
	firstInterfaceBool, e := utils.IsFirstInterface(netns, defaultConVeth)
	if e != nil {
		return fmt.Errorf("%s failed to check first veth interface: %v", logPrefix, e)
	}

	// 1. setup veth pair
	var hostInterface *current.Interface
	var conInterface *current.Interface
	if firstInterfaceBool {
		var err error
		hostInterface, conInterface, err = setupVeth(netns, args.ContainerID, prevResult)
		if err != nil {
			return fmt.Errorf("%s failed to setupVeth: %v", logPrefix, err)
		}
		fmt.Fprintf(os.Stderr, "%s succeeded to add first veth interface %s \n", logPrefix, defaultConVeth)
	} else {
		fmt.Fprintf(os.Stderr, "%s ingore to setup veth interface %s \n", logPrefix, defaultConVeth)
	}

	// get all ips on the node
	hostIPs, err := utils.GetHostIps()
	if err != nil {
		return fmt.Errorf("%s failed to GetHostIps: %v", logPrefix, err)
	}
	fmt.Fprintf(os.Stderr, "%s get host ip: %v \n", logPrefix, hostIPs)

	// get ips from pod
	conIPs, err := getChainedInterfaceIps(netns, preInterfaceName)
	if err != nil || len(conIPs) == 0 {
		return fmt.Errorf("%s failed to find ip from chained interface %s : %v", logPrefix, preInterfaceName, err)
	}
	fmt.Fprintf(os.Stderr, "%s get ip of chained interface %s : %v \n", logPrefix, preInterfaceName, conIPs)

	if enableIpv6 {
		if err := utils.EnableIpv6Sysctl(netns, defaultConVeth); err != nil {
			return fmt.Errorf("%s failed to enable ipv6 sysctl in pod ns(%v) : %v, firstInterfaceBool=%v", logPrefix, netns, err, firstInterfaceBool)
		}
	}

	// 2. setup neighborhood
	if err = setupNeighborhood(netns, hostInterface, conInterface, hostIPs, conIPs, args.ContainerID, firstInterfaceBool); err != nil {
		return fmt.Errorf("%s failed to setup neighbor: %v", logPrefix, err)
	}

	// 3. setup routes
	if err = setupRoutes(netns, hostInterface, conInterface, hostIPs, conIPs, conf.Routes, firstInterfaceBool, enableIpv4, enableIpv6); err != nil {
		return fmt.Errorf("%s failed to setup route: %v", logPrefix, err)
	}

	// 4. setup sysctl rp_filter
	if err = utils.SysctlRPFilter(netns, conf.RPFilter); err != nil {
		return fmt.Errorf("%s failed to setup sysctl: %v", logPrefix, err)
	}

	// TODO: for multiple macvlan interfaces, maybe need add "ip rule" for second interface

	fmt.Fprintf(os.Stderr, "%s succeeded to add veth interface for chained interface %s \n", logPrefix, preInterfaceName)
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
	for idx, route := range conf.Routes {
		if route.Dst.IP == nil {
			return nil, fmt.Errorf("[veth] routes[%v]: des must be specified", idx)
		}
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

	return &conf, nil
}

// setupVeth sets up a pair of virtual ethernet devices. It will create both veth
// devices and move the host-side veth into the provided hostNS namespace.
func setupVeth(netns ns.NetNS, containerID string, pr *current.Result) (*current.Interface, *current.Interface, error) {
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
func setupNeighborhood(netns ns.NetNS, hostInterface, chainedInterface *current.Interface, hostIPs, conIPs []string, containerID string, firstInterfaceBool bool) error {
	var err error
	// set neighborhood on host
	if err = neighborAdd(hostInterface.Name, chainedInterface.Mac, conIPs); err != nil {
		return err
	}

	if !firstInterfaceBool {
		return nil
	}

	// set up host veth Interface neighborhood in pod
	// bug?: sometimes hostveth's mac not be correct, we get hosVeth's mac via LinkByName
	hostVethLink, err := netlink.LinkByName(getHostVethName(containerID))
	if err != nil {
		return err
	}
	err = netns.Do(func(_ ns.NetNS) error {
		if err := neighborAdd(defaultConVeth, hostVethLink.Attrs().HardwareAddr.String(), hostIPs); err != nil {
			return err
		}
		return nil
	})

	return err
}

// setupRoutes setup routes for pod and host
// equivalent to: `ip route add $route`
func setupRoutes(netns ns.NetNS, hostInterface, chainedInterface *current.Interface, hostIPs, conIPs []string, routes []*types.Route, firstInterfaceBool, enableIpv4, enableIpv6 bool) error {
	var err error
	// set routes for host
	if err = utils.RouteAdd(hostInterface.Name, conIPs, enableIpv4, enableIpv6); err != nil {
		return err
	}

	if !firstInterfaceBool {
		return nil
	}

	// set routes for pod
	err = netns.Do(func(_ ns.NetNS) error {
		// add host ip route
		// equiva to "ip r add hostIP dev veth"
		if err = utils.RouteAdd(chainedInterface.Name, hostIPs, enableIpv4, enableIpv6); err != nil {
			return err
		}

		// setup custom routes from cni conf
		// such as calico cidr, service cidr
		link, err := netlink.LinkByName(chainedInterface.Name)
		if err != nil {
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

		for _, route := range routes {
			gw := net.IP{}
			if route.Dst.IP.To4() != nil {
				if destHostIpv4 != nil {
					gw = *destHostIpv4
				}
			} else {
				if destHostIpv6 != nil {
					gw = *destHostIpv6
				}
			}
			if len(gw) == 0 {
				return fmt.Errorf("[veth] failed to add route: %v: can't found next hop", route.Dst.String())
			}
			if err = netlink.RouteAdd(&netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       &route.Dst,
				Gw:        gw,
			}); err != nil {
				return fmt.Errorf("[veth] failed to add route: %v: %v ", route.Dst.String(), err)
			}
		}

		return nil
	})
	return err
}
