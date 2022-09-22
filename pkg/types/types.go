package types

import (
	"github.com/containernetworking/cni/pkg/types"
	"net"
)

type MigrateRoute int32

const (
	MigrateAuto MigrateRoute = iota - 1
	MigrateNever
	MigrateEnable
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

type LogOptions struct {
	LogLevel        string `json:"log_level"`
	LogFilePath     string `json:"log_file"`
	LogFileMaxSize  *int   `json:"log_max_size"`
	LogFileMaxAge   *int   `json:"log_max_age"`
	LogFileMaxCount *int   `json:"log_max_count"`
}

type Route struct {
	OverlaySubnet []string `json:"overlay_subnet,omitempty"`
	ServiceSubnet []string `json:"service_subnet,omitempty"`
	CustomSubnet  []string `json:"custom_subnet,omitempty"`
}
