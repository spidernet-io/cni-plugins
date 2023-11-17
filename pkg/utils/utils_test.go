package utils

import (
	"errors"
	"fmt"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"net/netip"
	"os"
	"regexp"
)

var _ = Describe("Utils", func() {

	Context("Test GetRuleNumber", Label("get-rule-number"), func() {

		It("first macvlan interface return table 100", func() {
			table := GetRuleNumber("net1")
			Expect(table).To(BeEquivalentTo(100))
		})

		It("second macvlan interface return table 101", func() {
			table := GetRuleNumber("net2")
			Expect(table).To(BeEquivalentTo(101))
		})

		It("input interface must be with prefix 'net*', or return -1 ", func() {
			table := GetRuleNumber("eth0")
			Expect(table).To(BeEquivalentTo(-1))
		})

	})

	Context("test GetChainedInterfaceIps", Label("get-ip"), func() {
		It("get ipv4-only ip by given interface name and netns", func() {
			ips, err := GetChainedInterfaceIps(testNetNs, conVethName, true, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(ips)).To(BeEquivalentTo(1))
			Expect(ips[0]).To(BeEquivalentTo(ipnets[0].String()))
		})

		It("get ipv6-only ip by given interface name and netns", func() {
			ips, err := GetChainedInterfaceIps(testNetNs, conVethName, false, true)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(ips)).To(BeEquivalentTo(1))
			Expect(ips[0]).To(BeEquivalentTo(ipnets[1].String()))
		})

		It("get ipv4/ipv6 ip by given interface name and netns", func() {
			ips, err := GetChainedInterfaceIps(testNetNs, conVethName, true, true)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(ips)).To(BeEquivalentTo(2))
			Expect(ips).To(BeEquivalentTo([]string{ipnets[0].String(), ipnets[1].String()}))
		})

		It("net interface err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.Interfaces, nil, errors.New("interface err"))
			_, err := GetChainedInterfaceIps(testNetNs, conVethName, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("parseCIDR err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.ParseCIDR, nil, nil, errors.New("parseCIDR err"))
			_, err := GetChainedInterfaceIps(testNetNs, conVethName, true, false)
			Expect(err).To(HaveOccurred())

		})
	})

	Context("test EnableIpv6Sysctl", Label("disable_ipv6"), func() {
		It("test set disable_ipv6 to 0", func() {
			err := EnableIpv6Sysctl(logging.LoggerFile, testNetNs)
			Expect(err).NotTo(HaveOccurred())

			// check disable_ipv6 = 0
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				dirs, err := os.ReadDir("/proc/sys/net/ipv6/conf")
				if err != nil {
					return err
				}

				for _, dir := range dirs {
					name := fmt.Sprintf("/net/ipv6/conf/%s/disable_ipv6", dir.Name())
					value, err := sysctl.Sysctl(name)
					Expect(err).NotTo(HaveOccurred())
					Expect(value).To(BeEquivalentTo("0"))
				}
				return err
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("os err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(os.ReadDir, nil, errors.New("os err"))
			err := EnableIpv6Sysctl(logging.LoggerFile, testNetNs)
			Expect(err).To(HaveOccurred())
		})

		It("sysctl err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(sysctl.Sysctl, nil, errors.New("sysctl err"))
			err := EnableIpv6Sysctl(logging.LoggerFile, testNetNs)
			Expect(err).To(HaveOccurred())
		})

		It("sysctl value return not 0", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(sysctl.Sysctl, "1", nil)
			err := EnableIpv6Sysctl(logging.LoggerFile, testNetNs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("sysctl2 value return not 0 err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncSeq(sysctl.Sysctl, []gomonkey.OutputCell{
				{Values: gomonkey.Params{"1", nil}},
				{Values: gomonkey.Params{"0", errors.New("sysctl err")}},
			})
			err := EnableIpv6Sysctl(logging.LoggerFile, testNetNs)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test ruleAdd/ruleDel", Label("rule"), func() {
		It("test add ipv4 rule table", func() {
			table := 200
			routes := []string{
				"2.2.2.0/24",
				"3.3.3.0/24",
			}

			_, net1, err := net.ParseCIDR("2.2.2.0/24")
			_, net2, err := net.ParseCIDR("3.3.3.0/24")
			Expect(err).NotTo(HaveOccurred())

			delNets := []netlink.Addr{
				{
					IPNet: net1,
				}, {
					IPNet: net2,
				},
			}

			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, true, false)
			})
			Expect(err).NotTo(HaveOccurred())

			// check if rule has been successfully added
			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V4)
				return err
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(2))

			for _, rule := range rules {
				found := rule.Dst.String() == routes[0] || rule.Dst.String() == routes[1]
				Expect(found).To(BeTrue())
			}

			// ruleDel
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(logger, table, delNets)
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("test add/del ipv6 rule table", func() {
			table := 201
			routes := []string{
				"fd00:22:6::/64",
			}

			_, net1, err := net.ParseCIDR("fd00:22:6::/64")
			Expect(err).NotTo(HaveOccurred())

			delNets := []netlink.Addr{
				{
					IPNet: net1,
				},
			}

			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, false, true)
			})
			Expect(err).NotTo(HaveOccurred())

			// check if rule has been successfully added
			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V6)
				return err
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(1))

			for _, rule := range rules {
				found := rule.Dst.String() == routes[0] || rule.Dst.String() == routes[1]
				Expect(found).To(BeTrue())
			}

			// ruleDel
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(logger, table, delNets)
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("test add/del ipv4/ipv6 rule table", func() {
			table := 202
			routes := []string{
				"4.4.4.0/24",
				"5.5.5.0/24",
				"fd00:44:6::/64",
				"fd00:55:6::/64",
			}

			_, net1, err := net.ParseCIDR("4.4.4.0/24")
			_, net2, err := net.ParseCIDR("5.5.5.0/24")
			_, net3, err := net.ParseCIDR("fd00:44:6::/64")
			_, net4, err := net.ParseCIDR("fd00:55:6::/64")
			Expect(err).NotTo(HaveOccurred())

			delNets := []netlink.Addr{
				{
					IPNet: net1,
				},
				{
					IPNet: net2,
				}, {
					IPNet: net3,
				},
				{
					IPNet: net4,
				},
			}

			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, true, true)
			})
			Expect(err).NotTo(HaveOccurred())

			// check if rule has been successfully added
			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_ALL)
				return err
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(4))

			for _, rule := range rules {
				found := rule.Dst.String() == routes[0] || rule.Dst.String() == routes[1] ||
					rule.Dst.String() == routes[2] || rule.Dst.String() == routes[3]
				Expect(found).To(BeTrue())
			}

			// rule del
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(logger, table, delNets)
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("invalid CIDR return error", func() {
			table := 203
			routes := []string{
				"abcd",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, true, true)
			})
			Expect(err).To(HaveOccurred())
		})

		It("only add routes with matching ipfamily and cidr", func() {
			table := 204
			routes := []string{
				"10.10.10.0/24",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, false, true)
			})
			Expect(err).NotTo(HaveOccurred())

			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V4)
				return err
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(0))
		})

		It("netlink ruleAdd err", func() {
			table := 200
			routes := []string{
				"2.2.2.0/24",
				"3.3.3.0/24",
			}
			err := testNetNs.Do(func(netNS ns.NetNS) error {
				patches := gomonkey.NewPatches()
				defer patches.Reset()
				patches.ApplyFuncReturn(netlink.RuleAdd, errors.New("rule add err"))
				return ruleAdd(logger, routes, table, true, false)
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test AddFromRuleTable", Label("from-rule"), func() {
		// eq: ip rule add from <ipnet> lookup <table>
		It("AddFromRuleTable for ipv4", func() {
			table := 207
			chainedIPs := []netlink.Addr{
				netlink.Addr{
					IPNet: &net.IPNet{
						IP:   net.ParseIP("10.6.212.204"),
						Mask: net.CIDRMask(24, 32),
					},
				},
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
				return AddFromRuleTable(logger, chainedIPs, table, true, false)
			})
			Expect(err).NotTo(HaveOccurred())

			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V4)
				return err
			})

			Expect(len(rules)).To(BeEquivalentTo(1))

			// del rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(logger, table, chainedIPs)
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("AddFromRuleTable for ipv6", func() {
			table := 208

			_, ipnet, err := net.ParseCIDR("fd00::1/96")
			Expect(err).NotTo(HaveOccurred())

			chainedIPs := []netlink.Addr{
				netlink.Addr{
					IPNet: ipnet,
				},
			}

			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return AddFromRuleTable(logger, chainedIPs, table, false, true)
			})
			Expect(err).NotTo(HaveOccurred())

			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V6)
				return err
			})

			Expect(len(rules)).To(BeEquivalentTo(1))

			// del rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(logger, table, chainedIPs)
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("AddFromRuleTable for dual", func() {
			table := 209

			_, ipnet, err := net.ParseCIDR("11.11.11.0/24")
			_, ipnet1, err := net.ParseCIDR("fd00:11:11:11::/64")
			Expect(err).NotTo(HaveOccurred())

			chainedIPs := []netlink.Addr{
				{
					IPNet: ipnet,
				}, {
					IPNet: ipnet1,
				},
			}

			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return AddFromRuleTable(logger, chainedIPs, table, true, true)
			})
			Expect(err).NotTo(HaveOccurred())

			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_ALL)
				return err
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(2))

			// del rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(logger, table, chainedIPs)
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("test RouteAdd", Label("route"), func() {
		It("test add ipv4 route", func() {
			// NOTE: netlink don't support list routes with non-main table, so we use unix.RT_TABLE_MAIN here
			table := unix.RT_TABLE_MAIN
			ips := []string{
				"10.10.10.10/24",
				"11.11.11.11/24",
			}
			_, _, err := RouteAdd(logger, table, defaultInterface, ips, true, false)
			Expect(err).NotTo(HaveOccurred())

			var routes []netlink.Route
			routes, err = routeList(defaultInterface, ips, table, netlink.FAMILY_V4)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(routes)).To(BeEquivalentTo(2))

			for _, route := range routes {
				// clean
				err = netlink.RouteDel(&route)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("test add ipv6 route", func() {
			// NOTE: netlink don't support list routes with non-main table, so we use unix.RT_TABLE_MAIN here
			table := unix.RT_TABLE_MAIN
			ips := []string{
				"fd00:10:10:10::10/64",
				"fd00:11:11:11::11/64",
			}
			_, _, err := RouteAdd(logger, table, defaultInterface, ips, false, true)
			Expect(err).NotTo(HaveOccurred())

			var routes []netlink.Route
			routes, err = routeList(defaultInterface, ips, table, netlink.FAMILY_V6)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(routes)).To(BeEquivalentTo(2))

			for _, route := range routes {
				// clean
				err = netlink.RouteDel(&route)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("test add ipv4/ipv6 route", func() {
			// NOTE: netlink don't support list routes with non-main table, so we use unix.RT_TABLE_MAIN here
			table := unix.RT_TABLE_MAIN
			ips := []string{
				"10.10.10.10/24",
				"fd00:10:10:10::10/64",
			}
			_, _, err := RouteAdd(logger, table, defaultInterface, ips, true, true)
			Expect(err).NotTo(HaveOccurred())

			var routes []netlink.Route
			routes, err = routeList(defaultInterface, ips, table, netlink.FAMILY_ALL)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(routes)).To(BeEquivalentTo(2))

			for _, route := range routes {
				// clean
				err = netlink.RouteDel(&route)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("test wrong input interface name", func() {
			// NOTE: netlink don't support list routes with non-main table, so we use unix.RT_TABLE_MAIN here
			table := unix.RT_TABLE_MAIN
			ips := []string{
				"10.10.10.10/24",
				"fd00:10:10:10::10/64",
			}

			wrongName := "ens1024"

			_, _, err := RouteAdd(logger, table, wrongName, ips, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("ips don't match ipfamily, skip add route", func() {
			table := 304
			ips := []string{
				"10.10.10.10/24",
			}

			_, _, err := RouteAdd(logger, table, defaultInterface, ips, false, true)
			Expect(err).NotTo(HaveOccurred())

			var routes []netlink.Route
			routes, err = routeList(defaultInterface, ips, table, netlink.FAMILY_V4)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(BeEmpty())
		})

		It("parseCIDR err", func() {
			// NOTE: netlink don't support list routes with non-main table, so we use unix.RT_TABLE_MAIN here
			table := unix.RT_TABLE_MAIN
			ips := []string{
				"10.10.10.10/24",
				"11.11.11.11/24",
			}
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.ParseCIDR, nil, nil, errors.New("parseCIDR err"))
			_, _, err := RouteAdd(logger, table, defaultInterface, ips, true, false)
			Expect(err).To(HaveOccurred())
		})

		It("route add err", func() {
			// NOTE: netlink don't support list routes with non-main table, so we use unix.RT_TABLE_MAIN here
			table := unix.RT_TABLE_MAIN
			ips := []string{
				"10.10.10.10/24",
				"11.11.11.11/24",
			}
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.RouteAdd, errors.New("parseCIDR err"))
			_, _, err := RouteAdd(logger, table, defaultInterface, ips, true, false)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test NeighborAdd", Label("neighbor"), func() {
		It("test add ipv4 neighbor table", func() {
			// add a neiborhood table in given netns
			testNetNs.Do(func(netNS ns.NetNS) error {
				err = NeighborAdd(logger, conVethName, hostInterface.HardwareAddr.String(), v4IP)
				Expect(err).NotTo(HaveOccurred())

				// check neighborhood table
				link, err := netlink.LinkByName(conVethName)
				Expect(err).NotTo(HaveOccurred())

				var neighs []netlink.Neigh
				neighs, err = netlink.NeighList(link.Attrs().Index, netlink.FAMILY_V4)
				Expect(err).NotTo(HaveOccurred())

				found := false
				netIP, _, err := net.ParseCIDR(v4IPStr)
				Expect(err).NotTo(HaveOccurred())

				var neighDst netlink.Neigh
				for _, neigh := range neighs {
					if neigh.HardwareAddr.String() == hostInterface.HardwareAddr.String() &&
						neigh.IP.String() == netIP.String() {
						neighDst = neigh
						found = true
					}
				}
				GinkgoWriter.Printf("found: %v\n", found)

				//clean
				err = netlink.NeighDel(&neighDst)
				Expect(err).NotTo(HaveOccurred())
				return nil
			})
		})

		It("wrong input interface name", func() {
			// add a neiborhood table in given netns
			testNetNs.Do(func(netNS ns.NetNS) error {
				err = NeighborAdd(logger, "tmp", hostInterface.HardwareAddr.String(), v4IP)
				Expect(err).To(HaveOccurred())
				return nil
			})
		})

		It("wrong input interface cidr", func() {
			// add a neiborhood table in given netns
			testNetNs.Do(func(netNS ns.NetNS) error {
				err = NeighborAdd(logger, conVethName, hostInterface.HardwareAddr.String(), net.IP{})
				Expect(err).To(HaveOccurred())
				return nil
			})
		})

		It("test add ipv4 neighbor add failed", func() {
			// add a neiborhood table in given netns
			testNetNs.Do(func(netNS ns.NetNS) error {
				patches := gomonkey.NewPatches()
				defer patches.Reset()
				patches.ApplyFuncReturn(netlink.NeighAdd, errors.New("NeighAdd failed"))
				err = NeighborAdd(logger, conVethName, hostInterface.HardwareAddr.String(), v4IP)
				Expect(err).To(HaveOccurred())
				return nil
			})
		})
	})

	Context("test GetHostIps", Label("getHostIPs"), func() {
		It("get host ipv4 ips", func() {
			ips, err := GetHostIps(logger, true, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(ips).NotTo(BeEmpty())

			// only ipv4 ip
			for _, ipStr := range ips {
				netip, _, err := net.ParseCIDR(ipStr)
				Expect(err).NotTo(HaveOccurred())
				Expect(netip.To4()).NotTo(BeNil())
			}
		})

		It("get host ipv6 ips", func() {
			_, err := GetHostIps(logger, false, true)
			Expect(err).To(HaveOccurred())
		})

		It("get host ipv4/ipv6 ips", func() {
			ips, err := GetHostIps(logger, true, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(ips).NotTo(BeEmpty())
		})

		It("net Interface err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.Interfaces, nil, errors.New("interface err"))
			_, err := GetHostIps(logger, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("regexp Compile err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(regexp.Compile, nil, errors.New("regexp err"))
			_, err := GetHostIps(logger, true, true)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test CheckInterfaceMiss", Label("check"), func() {
		It("return false if given interface exist", func() {
			exist, err := CheckInterfaceMiss(testNetNs, conVethName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exist).NotTo(BeTrue())
		})

		It("return true if given interface don't exist", func() {
			exist, err := CheckInterfaceMiss(testNetNs, "tmp-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(exist).To(BeTrue())
		})

	})

	Context("test AddrListByName", Label("addr"), func() {
		It("get all ips", func() {
			testNetNs.Do(func(netNS ns.NetNS) error {
				addrs, err := AddrListByName(conVethName, netlink.FAMILY_ALL)
				Expect(err).NotTo(HaveOccurred())
				Expect(addrs).NotTo(BeEmpty())
				return nil
			})
		})

		It("wrong input name", func() {
			testNetNs.Do(func(netNS ns.NetNS) error {
				_, err := AddrListByName("wrong", netlink.FAMILY_ALL)
				Expect(err).To(HaveOccurred())
				return nil
			})
		})

		It("AddrList failed", func() {
			testNetNs.Do(func(netNS ns.NetNS) error {
				patches := gomonkey.NewPatches()
				defer patches.Reset()
				patches.ApplyFuncReturn(netlink.AddrList, nil, errors.New("AddrList err"))
				_, err := AddrListByName(conVethName, netlink.FAMILY_ALL)
				Expect(err).To(HaveOccurred())
				return nil
			})
		})
	})

	Context("test SysctlRPFilter", func() {
		It("rp_filter value is 0,1,2", func() {
			for i := 0; i < 3; i++ {
				value0 := int32(i)
				enable := true
				rpFilter := &types.RPFilter{
					Enable: &enable,
					Value:  &value0,
				}
				err := SysctlRPFilter(logger, testNetNs, rpFilter)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("setRPFilter err", func() {
			for i := 0; i < 3; i++ {
				value0 := int32(i)
				enable := true
				rpFilter := &types.RPFilter{
					Enable: &enable,
					Value:  &value0,
				}
				patches := gomonkey.NewPatches()
				defer patches.Reset()
				patches.ApplyFuncReturn(setRPFilter, errors.New("setRPFilter err"))
				err := SysctlRPFilter(logger, testNetNs, rpFilter)
				Expect(err).To(HaveOccurred())
			}
		})

		It("netns setRPFilter err", func() {
			for i := 0; i < 3; i++ {
				value0 := int32(i)
				enable := false
				rpFilter := &types.RPFilter{
					Enable: &enable,
					Value:  &value0,
				}
				patches := gomonkey.NewPatches()
				defer patches.Reset()
				patches.ApplyFuncReturn(setRPFilter, errors.New("setRPFilter err"))
				err := SysctlRPFilter(logger, testNetNs, rpFilter)
				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("test HijackCustomSubnet", func() {
		It("overlay", func() {
			err := HijackCustomSubnet(logger, testNetNs, serviceSubnet, overlaySubnet, []string{}, defaultInterfaceAddrs, 100, true, true)
			Expect(err).NotTo(HaveOccurred())
		})
		It("underlay", func() {
			err := HijackCustomSubnet(logger, testNetNs, serviceSubnet, overlaySubnet, []string{}, defaultInterfaceAddrs, 101, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("overlay ruleAdd err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncSeq(ruleAdd, []gomonkey.OutputCell{
				{Values: gomonkey.Params{errors.New("rule add err")}},
				{Values: gomonkey.Params{nil}},
			})
			err := HijackCustomSubnet(logger, testNetNs, serviceSubnet, overlaySubnet, []string{}, defaultInterfaceAddrs, 100, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("overlay ruleAdd2 err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncSeq(ruleAdd, []gomonkey.OutputCell{
				{Values: gomonkey.Params{nil}},
				{Values: gomonkey.Params{errors.New("rule add err")}},
			})
			err := HijackCustomSubnet(logger, testNetNs, serviceSubnet, overlaySubnet, []string{}, defaultInterfaceAddrs, 100, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("underlay ruleAdd2 err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncSeq(ruleAdd, []gomonkey.OutputCell{
				{Values: gomonkey.Params{errors.New("rule add err")}},
				{Values: gomonkey.Params{nil}},
			})
			err := HijackCustomSubnet(logger, testNetNs, serviceSubnet, overlaySubnet, []string{}, defaultInterfaceAddrs, 101, true, true)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test MigrateRoute", func() {
		It("success MigrateRoute -1", func() {
			err := MigrateRoute(logger, testNetNs, conVethName, conVethName, defaultInterfaceAddrs, types.MigrateRoute(-1), 100, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("success MigrateRoute 0", func() {
			err := MigrateRoute(logger, testNetNs, conVethName, conVethName, defaultInterfaceAddrs, types.MigrateRoute(0), 100, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("success MigrateRoute -1 compare false", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(compareInterfaceName, false)
			err := MigrateRoute(logger, testNetNs, conVethName, conVethName, defaultInterfaceAddrs, types.MigrateRoute(-1), 100, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

	})
	Context("test AddToRuleTable", func() {
		It("overlay", func() {
			err := AddToRuleTable(logger, defaultInterfaceIPs, 100, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("parseCIDR err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.ParseCIDR, nil, nil, errors.New("parseCIDR err"))
			err := AddToRuleTable(logger, defaultInterfaceIPs, 100, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("netlink RuleAdd err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.RuleAdd, errors.New("netlink.RuleAdd err"))
			err := AddToRuleTable(logger, defaultInterfaceIPs, 100, true, true)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test GetDefaultRouteInterface", func() {
		It("net1", func() {
			ifc := GetDefaultRouteInterface("net1")
			Expect(ifc).To(Equal("eth0"))
		})
		It("net2", func() {
			ifc := GetDefaultRouteInterface("net2")
			Expect(ifc).To(Equal("net1"))
		})
	})

	Context("test GetNextHopIPs", func() {
		It("success", func() {
			_, err := GetNextHopIPs(logger, defaultInterfaceIPs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("parseCIDR", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.ParseCIDR, nil, nil, errors.New("parseCIDR err"))
			_, err := GetNextHopIPs(logger, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})

		It("netlink.RouteGet err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.RouteGet, nil, errors.New("route get err"))
			_, err := GetNextHopIPs(logger, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test compareInterfaceName", func() {
		It("true", func() {
			Expect(compareInterfaceName("net1", "eth0")).To(Equal(true))
		})
		It("false", func() {
			Expect(compareInterfaceName("eth0", "net1")).To(Equal(false))
		})
	})
	Context("test AddStaticNeighTable", func() {
		It("success", func() {
			err := AddStaticNeighTable(logger, testNetNs, false, conVethName, hostIPs, defaultInterfaceAddrs)
			Expect(err).NotTo(HaveOccurred())
		})
		It("skip", func() {
			err := AddStaticNeighTable(logger, testNetNs, true, conVethName, hostIPs, defaultInterfaceAddrs)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Test moveRouteTable", func() {
		It("success", func() {
			testNetNs.Do(func(netNS ns.NetNS) error {
				err := moveRouteTable(logger, conVethName, 100, 4)
				Expect(err).NotTo(HaveOccurred())
				return nil
			})

		})
	})

	Context("Test setRPFilter", func() {
		It("success", func() {
			var v *int32
			err := setRPFilter(logger, v)
			Expect(err).NotTo(HaveOccurred())
		})

		It("os err", func() {
			var v *int32
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(os.ReadDir, nil, errors.New("os err"))
			err := setRPFilter(logger, v)
			Expect(err).To(HaveOccurred())
		})

		It("sysctl err", func() {
			var v *int32
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(sysctl.Sysctl, nil, errors.New("sysctl err"))
			err := setRPFilter(logger, v)
			Expect(err).NotTo(HaveOccurred())
		})

		It("sysctl2 err", func() {
			var v *int32
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncSeq(sysctl.Sysctl, []gomonkey.OutputCell{
				{Values: gomonkey.Params{nil, nil}},
				{Values: gomonkey.Params{nil, errors.New("sysctl err")}},
			})
			err := setRPFilter(logger, v)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test convertIp2Mac", func() {
		regex, err := regexp.Compile("^" + "[a-fA-F0-9]{2}" + "(" + ":[a-fA-F0-9]{2}" + ")" + "{3}" + "$")
		Expect(err).To(BeNil())

		It("test ipv4", func() {
			addr, err := netip.ParseAddr("192.168.20.1")
			Expect(err).To(BeNil())

			macSufficx, err := inetAton(addr)
			Expect(err).To(BeNil())

			matched := regex.MatchString(macSufficx)
			Expect(matched).To(BeTrue())
		})

		It("test ipv6", func() {
			addr, err := netip.ParseAddr("fd00:110:120::1")
			Expect(err).To(BeNil())

			macSufficx, err := inetAton(addr)
			Expect(err).To(BeNil())

			matched := regex.MatchString(macSufficx)
			Expect(matched).To(BeTrue())
		})

		It("invalid ip", func() {
			_, err := inetAton(netip.Addr{})
			Expect(err.Error()).To(Equal("invalid ip address"))
		})
	})

	Context("Test OverwriteMacAddress", func() {

		It("a right config and pass", func() {
			newmac, err := OverwriteMacAddress(logger, testNetNs, "0a:1b", conVethName)
			Expect(err).NotTo(HaveOccurred())
			Expect(newmac).NotTo(BeEmpty(), newmac)
		})
	})

})
