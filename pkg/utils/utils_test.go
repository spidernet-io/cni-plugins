package utils

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
	"os"
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
	})

	Context("test ruleAdd/ruleDel", Label("rule"), func() {
		It("test add ipv4 rule table", func() {
			table := 200
			routes := []string{
				"2.2.2.0/24",
				"3.3.3.0/24",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
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
				return RuleDel(testNetNs, logger, table, routes)
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("test add/del ipv6 rule table", func() {
			table := 201
			routes := []string{
				"fd00:22:6::/64",
				"fd00:33:6::/64",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
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
			Expect(len(rules)).To(BeEquivalentTo(2))

			for _, rule := range rules {
				found := rule.Dst.String() == routes[0] || rule.Dst.String() == routes[1]
				Expect(found).To(BeTrue())
			}

			// ruleDel
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(testNetNs, logger, table, routes)
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

			err := testNetNs.Do(func(netNS ns.NetNS) error {
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
				return RuleDel(testNetNs, logger, table, routes)
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

		It("duplicate rule table can be added more than once", func() {
			table := 205
			routes := []string{
				"10.10.10.0/24",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, true, false)
			})
			Expect(err).NotTo(HaveOccurred())

			// add again
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, true, false)
			})
			Expect(err).NotTo(HaveOccurred())

			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V4)
				return err
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(2))

			// rule del
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(testNetNs, logger, table, routes)
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("ignore del non-exist rule table", func() {
			table := 206
			routes := []string{
				"10.10.10.0/24",
			}

			fake := []string{
				"30.30.30.0/24",
			}

			var err error
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return ruleAdd(logger, routes, table, true, false)
			})
			Expect(err).NotTo(HaveOccurred())

			var rules []netlink.Rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				rules, err = ruleList(table, netlink.FAMILY_V4)
				return err
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).To(BeEquivalentTo(1))

			// del non-exist rule
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(testNetNs, logger, table, fake)
			})
			Expect(err).NotTo(HaveOccurred())

			// clean
			err = testNetNs.Do(func(netNS ns.NetNS) error {
				return RuleDel(testNetNs, logger, table, routes)
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("test AddFromRuleTable", Label("from-rule"), func() {
		// eq: ip rule add from <ipnet> lookup <table>
		It("AddFromRuleTable for ipv4", func() {
			table := 207

			chainedIPs := []string{
				"10.10.10.0/24",
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
				return RuleDel(testNetNs, logger, table, chainedIPs)
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("AddFromRuleTable for ipv6", func() {
			table := 208

			chainedIPs := []string{
				"fd00:10:10:10::/64",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
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
				return RuleDel(testNetNs, logger, table, chainedIPs)
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("AddFromRuleTable for dual", func() {
			table := 209

			chainedIPs := []string{
				"11.11.11.0/24",
				"fd00:11:11:11::/64",
			}

			err := testNetNs.Do(func(netNS ns.NetNS) error {
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
				return RuleDel(testNetNs, logger, table, chainedIPs)
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
				netIP, _, err := net.ParseCIDR(v4IP)
				Expect(err).NotTo(HaveOccurred())

				var neighDst netlink.Neigh
				for _, neigh := range neighs {
					if neigh.HardwareAddr.String() == hostInterface.HardwareAddr.String() &&
						neigh.IP.String() == netIP.String() {
						neighDst = neigh
						found = true
					}
				}
				Expect(found).To(BeTrue())

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
				err = NeighborAdd(logger, conVethName, hostInterface.HardwareAddr.String(), "wrong")
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

})
