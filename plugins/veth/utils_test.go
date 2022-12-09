package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net"
)

var _ = Describe("Veth utils", func() {
	defer GinkgoRecover()
	Context("Test filterIPs", func() {

		It("only one ipv4 ip", func() {
			ips := []string{"192.168.1.1"}
			ipv4, ipv6 := false, false
			for _, ip := range ips {
				got := []string{}
				netIP := net.ParseIP(ip)
				ipv4, ipv6, got = filterIPs(netIP, ipv4, ipv6, got)
				Expect(got).To(Equal([]string{ip}))
			}
		})

		It("ipv4 and ipv6 ip", func() {
			ips := []string{"192.168.1.1", "fd00:1033::f197:b232:eaa:bac0"}
			ipv4, ipv6 := false, false
			for _, ip := range ips {
				got := []string{}
				netIP := net.ParseIP(ip)
				ipv4, ipv6, got = filterIPs(netIP, ipv4, ipv6, got)
				Expect(got).To(Equal([]string{ip}))
			}
		})

	})
})
