package common

import (
	"fmt"
	"github.com/spidernet-io/e2eframework/framework"
	corev1 "k8s.io/api/core/v1"
	"log"
	"time"
)

func GetServiceNodePorts(ports []corev1.ServicePort) []int32 {
	nodePorts := make([]int32, 0, len(ports))
	for _, port := range ports {
		nodePorts = append(nodePorts, port.NodePort)
	}
	return nodePorts
}

func WaitEndpointReady(retryTimes int, name, namespace string, frame *framework.Framework) error {
	var err error
	var period = 500 * time.Millisecond
	for ; retryTimes > 0; retryTimes-- {
		_, err = frame.GetEndpoint(name, namespace)
		if err == nil {
			return nil
		}
		period = 2 * period
		time.Sleep(period)
	}
	log.Println("retryTime: ", retryTimes)
	return fmt.Errorf("failed to get endpoint %s/%s with retry %d times", namespace, name, retryTimes)
}
