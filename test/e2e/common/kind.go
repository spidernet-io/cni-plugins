package common

import (
	"context"
	"fmt"
	e2e "github.com/spidernet-io/e2eframework/framework"
	_ "github.com/spidernet-io/spiderpool/test/e2e/common"
	"net"
	"strings"
)

func GetKindNodeIPs(ctx context.Context, f *e2e.Framework, nodeList []string) ([]string, error) {
	var nodeIPCIDRs []string
	command := fmt.Sprintf("ip addr show %s | grep inet | grep global | awk '{print $2}'", KindNodeDefaultInterface)
	for _, node := range nodeList {
		output, err := f.DockerExecCommand(ctx, node, command)
		if err != nil {
			return nil, err
		}
		nodeIPCIDRs = append(nodeIPCIDRs, strings.Split(strings.TrimSpace(string(output)), "\n")...)
	}

	var nodeIPs []string
	for _, nodeIPCIDR := range nodeIPCIDRs {
		nodeIP, _, err := net.ParseCIDR(nodeIPCIDR)
		if err != nil {
			return nil, err
		}
		if nodeIP.To4() != nil && IPV4 {
			nodeIPs = append(nodeIPs, nodeIP.String())
		}
		if nodeIP.To4() == nil && IPV6 {
			nodeIPs = append(nodeIPs, nodeIP.String())
		}
	}

	return nodeIPs, nil
}
