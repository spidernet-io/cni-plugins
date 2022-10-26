package common

import "time"

// multus crd type
const (
	MULTUS_DEFAULT = iota
	MULTUS_MACVLAN_STANDALONE
	MULTUS_MACVLAN_OVERLAY
)

// multus crd name
var (
	MacvlanStandaloneVlan100Name = "macvlan-standalone-vlan0"
	MacvlanStandaloneVlan200Name = "macvlan-standalone-vlan100"
	MacvlanOverlayVlan100Name    = "macvlan-overlay-vlan0"
	MacvlanOverlayVlan200Name    = "macvlan-overlay-vlan100"
)

// annotations
var (
	MultusDefaultAnnotationKey     = "v1.multus-cni.io/default-network"
	MultusAddonAnnotation_Key      = "k8s.v1.cni.cncf.io/networks"
	SpiderPoolIPPoolAnnotationKey  = "ipam.spidernet.io/ippool"
	SpiderPoolIPPoolsAnnotationKey = "ipam.spidernet.io/ippools"
)

var (
	KindNodeDefaultInterface   = "eth0"
	CtxTimeout                 = 60 * time.Second
	ENV_VLAN_GATEWAY_CONTAINER = "VLAN_GATEWAY_CONTAINER"
)
