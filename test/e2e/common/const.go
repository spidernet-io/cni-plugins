package common

import (
	"os"
	"time"
)

// multus crd type
const (
	MULTUS_DEFAULT = iota
	MULTUS_MACVLAN_STANDALONE
	MULTUS_MACVLAN_OVERLAY
)

// multus crd name
var (
	MacvlanStandaloneVlan0Name   = "macvlan-standalone-vlan0"
	MacvlanStandaloneVlan100Name = "macvlan-standalone-vlan100"
	MacvlanOverlayVlan0Name      = "macvlan-overlay-vlan0"
	MacvlanOverlayVlan100Name    = "macvlan-overlay-vlan100"
)

// annotations
var (
	MultusDefaultAnnotationKey     = "v1.multus-cni.io/default-network"
	MultusAddonAnnotation_Key      = "k8s.v1.cni.cncf.io/networks"
	SpiderPoolIPPoolAnnotationKey  = "ipam.spidernet.io/ippool"
	SpiderPoolIPPoolsAnnotationKey = "ipam.spidernet.io/ippools"
	SpiderPoolIPAnnotationsKey     = []string{
		"ipam.spidernet.io/assigned-net1",
		"ipam.spidernet.io/assigned-net2",
		"ipam.spidernet.io/assigned-eth0",
	}
)

var (
	KindNodeDefaultInterface   = "eth0"
	CtxTimeout                 = 60 * time.Second
	ENV_VLAN_GATEWAY_CONTAINER = "VLAN_GATEWAY_CONTAINER"
)
var (
	IPV4       = true
	IPV6       = true
	TestMultus = true
)

var (
	MultusNs                = "kube-system"
	SpiderDoctorAgentNs     = "kube-system"
	SpiderDoctorAgentDSName = "spiderdoctor-agent"
)

func init() {
	IPV4 = os.Getenv("E2E_IPV4_ENABLED") == "true"
	IPV6 = os.Getenv("E2E_IPV6_ENABLED") == "true"
	// https://github.com/spidernet-io/cni-plugins/issues/143
	TestMultus = os.Getenv("DEFAULT_CNI") != "cilium"
}
