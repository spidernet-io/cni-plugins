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

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Suite")
}

var hostInterface, conInterface net.Interface
var testNetNs ns.NetNS
var logger *zap.Logger
var logPath = "/tmp/meta-plugins/tmp.log"
var containerID = "testtesttesttest"
var defaultInterface = "eth0"
var hostVethName, secondifName, overlayifName, v4IP, v6IP, v4IP2, v6IP2 string
var err error
var hostIPs []net.IP
var defaultInterfaceIPs []netlink.Addr

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
	//v4IP = "10.6.212.100/16"
	//v6IP = "fd00:10:6:212::100/64"
	//v4IP2 = "10.6.172.100/16"
	//v6IP2 = "fd00:10:6:172::100/64"

	v4IP = "172.17.2.0/16"
	v6IP = "fd00:172:17:2::100/64"
	v4IP2 = "172.17.2.0/16"
	v6IP2 = "fd00:172:17:2::100/64"
	ipnets := generateIPNet(v4IP, v6IP)
	ipnets2 := generateIPNet(v4IP2, v6IP2)

	hostIPs = append(hostIPs, net.ParseIP("172.17.2.100"))
	hostIPs = append(hostIPs, net.ParseIP("fd00:172:17:2::100"))

	defaultInterfaceIPs = []netlink.Addr{
		{
			IPNet: &net.IPNet{
				IP:   net.ParseIP("10.6.212.204"),
				Mask: net.CIDRMask(24, 32),
			},
		},
	}

	secondifName = "net1"
	overlayifName = "eth0"
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
		hostInterface, conInterface, err = ip.SetupVethWithName(overlayifName, hostVethName, 1500, "", hostNS)
		Expect(err).NotTo(HaveOccurred())

		err = netlink.LinkAdd(&netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name: secondifName,
			},
		})
		overlaylink, err := netlink.LinkByName(overlayifName)
		Expect(err).NotTo(HaveOccurred())

		secondlink, err := netlink.LinkByName(secondifName)
		Expect(err).NotTo(HaveOccurred())

		err = netlink.LinkSetUp(secondlink)
		Expect(err).NotTo(HaveOccurred())

		err = netlink.LinkSetUp(overlaylink)
		Expect(err).NotTo(HaveOccurred())

		err = utils.EnableIpv6Sysctl(logger, testNetNs)
		Expect(err).NotTo(HaveOccurred())

		for _, ipnet := range ipnets {
			err = netlink.AddrAdd(overlaylink, &netlink.Addr{IPNet: ipnet})
			Expect(err).NotTo(HaveOccurred())
		}

		for _, ipnet := range ipnets2 {
			err = netlink.AddrAdd(secondlink, &netlink.Addr{IPNet: ipnet})
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
	//os.RemoveAll(logPath)
})
