package types

import (
	"github.com/containernetworking/cni/pkg/types"
	"net"
)

type RPFilter struct {
	// setup host rp_filter
	Enable *bool `json:"set_host,omitempty"`
	// the value of rp_filter, must be 0/1/2
	Value *int32 `json:"value,omitempty"`
}

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString //revive:disable-line
	K8S_POD_NAMESPACE          types.UnmarshallableString //revive:disable-line
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString //revive:disable-line
	K8S_POD_UID                types.UnmarshallableString //revive:disable-line
}
