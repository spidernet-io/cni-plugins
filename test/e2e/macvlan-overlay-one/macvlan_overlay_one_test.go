package macvlan_overlay_one_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/test/e2e/common"
	"net"
	"os"
)

var _ = Describe("MacvlanOverlayOne", Label("overlay", "one-nic"), func() {
	It("Host can be communicate with pod, including pod in same and different node", Label("ping"), func() {
		for _, node := range frame.Info.KindNodeList {
			for _, podIP := range podIPs {
				command := common.GetPingCommandByIPFamily(podIP)
				_, err := frame.DockerExecCommand(context.TODO(), node, command)
				Expect(err).NotTo(HaveOccurred(), " host %s failed to ping %s: %v,", node, podIP, err)
			}
		}
	})

	It("Pods in different node can be communicate with each other", Label("ping"), func() {
		for _, pod := range podList.Items {
			for _, podIP := range podIPs {
				command := common.GetPingCommandByIPFamily(podIP)
				_, err := frame.ExecCommandInPod(pod.Name, pod.Namespace, command, context.TODO())
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("pod %s/%s failed to %s", pod.Namespace, pod.Name, command))
			}
		}
	})

	// ping calico pod ?

	It("remote's client should be access to pod", Label("ping"), func() {
		// get remote's client container name
		vlanGatewayName := os.Getenv(common.ENV_VLAN_GATEWAY_CONTAINER)
		Expect(vlanGatewayName).NotTo(BeEmpty(), "failed to get env 'VLAN_GATEWAY_CONTAINER'")

		for _, podIP := range podIPs {
			command := common.GetPingCommandByIPFamily(podIP)
			_, err := frame.DockerExecCommand(context.TODO(), vlanGatewayName, command)
			Expect(err).To(Succeed())
		}
	})

	It("Pod should be access to clusterIP, including ipv4 and ipv6", Label("curl"), func() {
		for _, pod := range podList.Items {
			for _, clusterIP := range clusterIPs {
				// TODO https://github.com/projectcalico/calico/issues/6877
				if net.ParseIP(clusterIP).To4() == nil {
					continue
				}
				command := common.GetCurlCommandByIPFamily(clusterIP, 80)
				output, err := frame.ExecCommandInPod(pod.Name, pod.Namespace, command, context.TODO())
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("pod %s/%s failed to %s: %s \n", pod.Namespace, pod.Name, command, output))
			}
		}
	})

	It("Host should be access to clusterIP, including ipv4 and ipv6", Label("curl"), func() {
		for _, node := range frame.Info.KindNodeList {
			for _, clusterIP := range clusterIPs {
				// TODO https://github.com/projectcalico/calico/issues/6877
				if net.ParseIP(clusterIP).To4() == nil {
					continue
				}
				command := common.GetCurlCommandByIPFamily(clusterIP, 80)
				GinkgoWriter.Printf("docker exec -it %s %s \n", node, command)
				output, err := frame.DockerExecCommand(context.TODO(), node, command)
				Expect(err).NotTo(HaveOccurred(), " host %s failed to access to cluster service %s: %s,", node, clusterIP, output)
			}
		}

	})
	It("Host should be access to nodePort address, including ipv4 and ipv6", Label("curl"), func() {
		for _, node := range frame.Info.KindNodeList {
			for _, nodeIP := range nodeIPs {
				// TODO https://github.com/projectcalico/calico/issues/6877
				if net.ParseIP(nodeIP).To4() == nil {
					continue
				}
				command := common.GetCurlCommandByIPFamily(nodeIP, nodePorts[0])
				GinkgoWriter.Printf("docker exec -it %s %s \n", node, command)
				output, err := frame.DockerExecCommand(context.TODO(), node, command)
				Expect(err).NotTo(HaveOccurred(), " host %s failed to access to nodePort service %s/%d: %s \n", node, nodeIP, nodePorts[0], output)
			}
		}
	})
})
