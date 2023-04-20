package config

import (
	"fmt"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"k8s.io/utils/pointer"
	"net"
	"regexp"
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

func ValidateOverwriteMacAddress(prefix string) error {
	if prefix == "" {
		return nil
	}
	// prefix format like: 00:00、0a:1b
	matchRegexp, err := regexp.Compile("^" + "(" + "[a-fA-F0-9]{2}[:-][a-fA-F0-9]{2}" + ")" + "$")
	if err != nil {
		return err
	}
	if !matchRegexp.MatchString(prefix) {
		return fmt.Errorf("mac_prefix format should be match regex: [a-fA-F0-9]{2}[:][a-fA-F0-9]{2}, like '0a:1b'")
	}
	return nil
}

func ValidateIPConflict(config *ty.IPConflict) *ty.IPConflict {
	if config == nil {
		return nil
	}
	if config.Enabled {
		if config.Interval == "" {
			config.Interval = "1s"
		}

		if config.Retry <= 0 {
			config.Retry = 3
		}
	}
	return config

}
