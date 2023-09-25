package utils

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/e2eframework/tools"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"net"
	"os"
	"syscall"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var hostInterface, conInterface net.Interface
var testNetNs ns.NetNS
var logger *zap.Logger
var ipnets [2]*net.IPNet
var serviceSubnet, overlaySubnet = []string{"10.96.0.0/16", "fd00:10:96::/112"}, []string{"10.244.0.0/16", "fd00:10:244::/56"}
var defaultInterfaceIPs = []string{"10.96.0.12/24"}
var defaultInterfaceAddrs []netlink.Addr

// change me, default value is eth0 on github runner
var defaultInterface = "eth0"
var conVethName, hostVethName, v4IPStr, v6IPStr, logPath string
var v4IP, v6IP net.IP
var hostIPs []net.IP
var err error

func generateRandomName() string {
	return fmt.Sprintf("veth%s", tools.RandomName()[8:])
}

func ruleList(table, ipfamily int) ([]netlink.Rule, error) {
	rules, err := netlink.RuleList(ipfamily)
	if err != nil {
		return nil, err
	}

	filterRules := make([]netlink.Rule, 0, len(rules))
	for _, rule := range rules {
		if rule.Table == table {
			filterRules = append(filterRules, rule)
		}
	}
	return filterRules, nil
}

func routeList(iface string, ips []string, table, ipfamily int) ([]netlink.Route, error) {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return nil, err
	}

	filterIPs := make([]net.IP, 0, len(ips))
	for _, ipStr := range ips {
		netip, _, err := net.ParseCIDR(ipStr)
		if err != nil {
			return nil, err
		}
		filterIPs = append(filterIPs, netip)
	}

	filterRoute := make([]netlink.Route, 0)
	routes, err := netlink.RouteList(link, ipfamily)
	for _, route := range routes {
		for _, filterIP := range filterIPs {
			if route.Dst != nil && route.Dst.IP.String() == filterIP.String() {
				filterRoute = append(filterRoute, route)
			}
		}
	}
	if err != nil {
		return nil, err
	}

	return filterRoute, nil
}

var _ = BeforeSuite(func() {

	conVethName = generateRandomName()
	hostVethName = generateRandomName()
	v4IPStr = "10.6.212.100/16"
	v6IPStr = "fd00:10:6:212::100/64"
	logPath = "/tmp/meta-plugins/tmp.log"
	v4IP, ipnets[0], _ = net.ParseCIDR(v4IPStr)
	v6IP, ipnets[1], _ = net.ParseCIDR(v6IPStr)
	hostIPs = append(hostIPs, v4IP)

	if logging.LoggerFile == nil {
		logOptions := logging.InitLogOptions(&types.LogOptions{LogFilePath: logPath})
		err := logging.SetLogOptions(logOptions)
		Expect(err).NotTo(HaveOccurred())
	}

	_, netIPs, _ := net.ParseCIDR("10.96.0.12/24")
	defaultInterfaceAddrs = []netlink.Addr{
		{
			IPNet: netIPs,
		},
	}
	logger = logging.LoggerFile.Named("unit-test")

	// create net ns
	testNetNs, err = testutils.NewNS()
	Expect(err).NotTo(HaveOccurred())
	// add test ip
	testNetNs.Do(func(hostNS ns.NetNS) error {
		// add test ip
		hostInterface, conInterface, err = ip.SetupVethWithName(conVethName, hostVethName, 1500, "", hostNS)
		Expect(err).NotTo(HaveOccurred())

		link, err := netlink.LinkByName(conVethName)
		Expect(err).NotTo(HaveOccurred())

		err = netlink.LinkSetUp(link)
		Expect(err).NotTo(HaveOccurred())

		err = EnableIpv6Sysctl(logger, testNetNs)
		Expect(err).NotTo(HaveOccurred())

		for _, ipnet := range ipnets {
			err = netlink.AddrAdd(link, &netlink.Addr{IPNet: ipnet})
			Expect(err).NotTo(HaveOccurred())
		}
		return nil
	})

})

var _ = AfterSuite(func() {
	// clean ns
	if testNetNs != nil {
		testNetNs.Close()
		err := syscall.Unmount(testNetNs.Path(), syscall.MNT_DETACH)
		Expect(err).NotTo(HaveOccurred())
		os.RemoveAll(testNetNs.Path())
	}
	os.RemoveAll(logPath)
})
