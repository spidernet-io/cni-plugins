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

var testNetNs ns.NetNS
var logger *zap.Logger
var ipnets [2]*net.IPNet
var conVethName, hostVethName, v4IP, v6IP, logPath string
var err error

func generateIPNet(ipv4, ipv6 string) (ipnets [2]*net.IPNet) {

	_, ipnets[0], err = net.ParseCIDR(ipv4)
	Expect(err).NotTo(HaveOccurred())

	_, ipnets[1], err = net.ParseCIDR(ipv6)
	Expect(err).NotTo(HaveOccurred())

	return
}

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

var _ = BeforeSuite(func() {

	conVethName = generateRandomName()
	hostVethName = generateRandomName()
	v4IP = "10.6.212.100/16"
	v6IP = "fd00:10:6:212::100/64"
	logPath = "/tmp/meta-plugins/tmp.log"
	ipnets = generateIPNet(v4IP, v6IP)

	if logging.LoggerFile == nil {
		logOptions := logging.InitLogOptions(&types.LogOptions{LogFilePath: logPath})
		err := logging.SetLogOptions(logOptions)
		Expect(err).NotTo(HaveOccurred())
	}
	logger = logging.LoggerFile.Named("unit-test")

	// create net ns
	testNetNs, err = testutils.NewNS()
	Expect(err).NotTo(HaveOccurred())

	// add test ip
	testNetNs.Do(func(hostNS ns.NetNS) error {
		// add test ip
		_, _, err := ip.SetupVethWithName(conVethName, hostVethName, 1500, "", hostNS)
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
