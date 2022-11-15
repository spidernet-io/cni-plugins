package utils

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/vishvananda/netlink"
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

})
