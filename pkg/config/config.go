package config

import (
	"fmt"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"k8s.io/utils/pointer"
	"net"
	"strings"
)

func ValidateRPFilterConfig(config *ty.RPFilter) *ty.RPFilter {
	if config == nil {
		return &ty.RPFilter{
			Enable: pointer.Bool(true),
			Value:  pointer.Int32(2),
		}
	}

	if config.Enable != nil && *config.Enable {
		if config.Value != nil {
			matched := false
			for _, value := range []int32{0, 1, 2} {
				if *config.Value == value {
					matched = true
				}
			}
			if !matched {
				config.Value = pointer.Int32(2)
			}
		} else {
			config.Value = pointer.Int32(2)
		}
	}
	return config
}

func ValidateMigrateRouteConfig(given *ty.MigrateRoute) *ty.MigrateRoute {
	found := false
	if given == nil {
		return (*ty.MigrateRoute)(pointer.Int32(-1))
	}
	for _, value := range []ty.MigrateRoute{ty.MigrateAuto, ty.MigrateNever, ty.MigrateEnable} {
		if value == *given {
			found = true
		}
	}
	if !found {
		return (*ty.MigrateRoute)(pointer.Int32(-1))
	}
	return given
}

func ValidateRoutes(overlaySubnet, serviceSubnet []string) (ovlSubnet, svcSubnet []string, err error) {
	if len(overlaySubnet) == 0 {
		return nil, nil, fmt.Errorf("the subnet of overlay cni(such as calico or cilium) must be given")
	}
	if len(serviceSubnet) == 0 {
		return nil, nil, fmt.Errorf("the subnet of service clusterip must be given")
	}

	ovlSubnet, err = validateRoutes(overlaySubnet)
	if err != nil {
		return nil, nil, err
	}
	svcSubnet, err = validateRoutes(serviceSubnet)
	if err != nil {
		return nil, nil, err
	}

	return ovlSubnet, svcSubnet, nil
}

func validateRoutes(routes []string) ([]string, error) {
	result := make([]string, len(routes))
	for idx, route := range routes {
		result[idx] = strings.TrimSpace(route)
	}
	for _, route := range result {
		_, _, err := net.ParseCIDR(route)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
