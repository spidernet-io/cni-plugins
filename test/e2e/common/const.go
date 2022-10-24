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
	MacvlanStandaloneVlan100Name = "macvlan-standalone-vlan100"
	MacvlanStandaloneVlan200Name = "macvlan-standalone-vlan200"
	MacvlanOverlayVlan100Name    = "macvlan-overlay-vlan100"
	MacvlanOverlayVlan200Name    = "macvlan-overlay-vlan200"
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
