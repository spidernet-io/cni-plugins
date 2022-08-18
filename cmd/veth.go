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
	"github.com/vishvananda/netlink"
	"k8s.io/utils/pointer"
	"net"
	"runtime"
)

type RPFilterValue int

const (
	RPFilterDISABLE RPFilterValue = iota
	RPFilterStrict
	RPFilterLoose
)

type RPFilter struct {
	// setup host rp_filter
	Enable *bool `json:"enable,omitempty"`
	// the value of rp_filter, must be 0/1/2
	Value *RPFilterValue `json:"value,omitempty"`
}

type PluginConf struct {
	types.NetConf
	Routes []*types.Route
	// RpFilter
	RPFilter *RPFilter `json:"rp_filter,omitempty"`
	Skipped  bool      `json:"skip,omitempty"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
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
		return nil
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

	// Check if the veth-plugin has been executed
	// if so, skip it
	if isSkipped(netns) {
		return nil
	}

	// 1. setup veth pair
	hostInterface, conInterface, err := setupVeth(netns, args.ContainerID, prevResult)
	if err != nil {
		return err
	}

	// get all ips on the node
	hostIPs, err := getHostIps()
	if err != nil {
		return err
	}
	// get ips from pod eth0
	conIPs, err := getConIps(netns)
	if err != nil {
		return err
	}

	// 2. setup neighborhood
	if err = setupNeighborhood(netns, hostInterface, conInterface, hostIPs, conIPs); err != nil {
		return err
	}

	// 3. setup routes
	if err = setupRoutes(netns, hostInterface, conInterface, hostIPs, conIPs, conf.Routes); err != nil {
		return err
	}

	// 4. setup sysctl rp_filter
	if err = sysctlRPFilter(netns, conf.RPFilter); err != nil {
		return err
	}
	return nil
}

func cmdDel(args *skel.CmdArgs) error {
	// TODO
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
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	// Parse previous result. This will parse, validate, and place the
	// previous result object into conf.PrevResult. If you need to modify
	// or inspect the PrevResult you will need to convert it to a concrete
	// versioned Result struct.
	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("could not parse prevResult: %v", err)
	}
	// End previous result parsing

	// some validation
	for idx, route := range conf.Routes {
		if route.Dst.IP == nil {
			return nil, fmt.Errorf("routes[%v]: des must be specified", idx)
		}
	}

	// value must be 0/1/2
	// If not, giving default value: RPFilter_Loose(2) to it
	if conf.RPFilter != nil {
		if conf.RPFilter.Enable != nil && *conf.RPFilter.Enable {
			matched := false
			for _, value := range []RPFilterValue{RPFilterDISABLE, RPFilterStrict, RPFilterLoose} {
				if *conf.RPFilter.Value == value {
					matched = true
				}
			}
			if !matched {
				conf.RPFilter.Value = rpValue(RPFilterLoose)
			}
		}
	} else {
		// give default value: RPFilter_Loose(2)
		conf.RPFilter = &RPFilter{
			Enable: pointer.Bool(true),
			Value:  rpValue(RPFilterLoose),
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
			return err
		}
		hostInterface.Name = hostVeth.Name
		hostInterface.Mac = hostVeth.HardwareAddr.String()
		containerInterface.Name = contVeth0.Name
		containerInterface.Mac = contVeth0.HardwareAddr.String()
		containerInterface.Sandbox = netns.Path()

		pr.Interfaces = append(pr.Interfaces, hostInterface, containerInterface)

		if err = setLinkup(contVeth0.Name); err != nil {
			return err
		}
		return nil
	})
	if err = setLinkup(hostInterface.Name); err != nil {
		return nil, nil, err
	}

	return hostInterface, containerInterface, nil
}

// setupNeighborhood setup neighborhood tables for pod and host.
// equivalent to: `ip neigh add ....`
func setupNeighborhood(netns ns.NetNS, hostInterface, conInterface *current.Interface, hostIPs, conIPs []string) error {
	var err error
	// set neighborhood on host
	if err = neiAdd(hostInterface.Name, conInterface.Mac, conIPs); err != nil {
		return err
	}
	// set up neighborhood in pod
	err = netns.Do(func(_ ns.NetNS) error {
		if err := neiAdd(defaultConVeth, hostInterface.Mac, hostIPs); err != nil {
			return err
		}
		return nil
	})

	return err
}

// setupRoutes setup routes for pod and host
// equivalent to: `ip route add $route
func setupRoutes(netns ns.NetNS, hostInterface, conInterface *current.Interface, hostIPs, conIPs []string, routes []*types.Route) error {
	var err error
	// set routes for host
	if err = routeAdd(hostInterface.Name, conIPs); err != nil {
		return err
	}

	// set routes for pod
	err = netns.Do(func(_ ns.NetNS) error {
		// add host ip route
		// equiva to "ip r add hostIP dev veth"
		if err = routeAdd(conInterface.Name, hostIPs); err != nil {
			return err
		}

		// setup custom routes from cni conf
		// such as calico cidr, service cidr
		link, err := netlink.LinkByName(conInterface.Name)
		if err != nil {
			return err
		}

		// get one host ipv4 ip and one host ipv6 ip(if exist)
		ipv4, ipv6 := false, false
		viaIPs := make([]string, 0, 2)
		for _, ip := range hostIPs {
			netIP := net.ParseIP(ip)
			ipv4, ipv6, viaIPs = filterIPs(netIP, ipv4, ipv6, viaIPs)
		}

		ipMap := make(map[string]net.IP)
		for _, viaIP := range viaIPs {
			netIP := net.ParseIP(viaIP)
			if netIP.To4() != nil {
				ipMap["ipv4"] = netIP
			} else {
				ipMap["ipv6"] = netIP
			}
		}

		for _, route := range routes {
			gw := net.IP{}
			if route.Dst.IP.To4() != nil {
				if _, ok := ipMap["ipv4"]; ok {
					gw = ipMap["ipv4"]
				}
			} else {
				if _, ok := ipMap["ipv6"]; ok {
					gw = ipMap["ipv6"]
				}
			}
			if len(gw) == 0 {
				return fmt.Errorf("[veth]add route: %v failed: can't found next hop", route.Dst.String())
			}
			if err = netlink.RouteAdd(&netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       &route.Dst,
				Gw:        gw,
			}); err != nil {
				return fmt.Errorf("[veth]add route: %v failed: %v ", route.Dst.String(), err)
			}
		}

		return nil
	})
	return err
}

// setSysctlRp set rp_filter value
func sysctlRPFilter(netns ns.NetNS, rp *RPFilter) error {
	var err error
	// set host rp_filter
	if *rp.Enable {
		if err = setRPFilter(); err != nil {
			return err
		}
	}
	// set pod rp_filter
	err = netns.Do(func(_ ns.NetNS) error {
		if err := setRPFilter(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
