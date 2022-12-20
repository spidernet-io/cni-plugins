package main

import (
	"fmt"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
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

func TestVeth(t *testing.T) {
	defer GinkgoRecover()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Veth Suite")
}

var hostInterface, conInterface net.Interface
var testNetNs ns.NetNS
var logger *zap.Logger
var logPath = "/tmp/meta-plugins/tmp.log"
var containerID = "testtesttesttest"
var defaultInterface = "eth0"
var conVethName, hostVethName, v4IP, v6IP string
var err error
var defaultInterfaceIPs = []string{"10.96.0.12/24"}

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

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	var err error
	v4IP = "10.6.212.100/16"
	v6IP = "fd00:10:6:212::100/64"
	ipnets := generateIPNet(v4IP, v6IP)
	conVethName = "net1"
	hostVethName = generateRandomName()
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
		hostInterface, conInterface, err = ip.SetupVethWithName(conVethName, hostVethName, 1500, "", hostNS)
		Expect(err).NotTo(HaveOccurred())

		link, err := netlink.LinkByName(conVethName)
		Expect(err).NotTo(HaveOccurred())

		err = netlink.LinkSetUp(link)
		Expect(err).NotTo(HaveOccurred())

		err = utils.EnableIpv6Sysctl(logger, testNetNs)
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
