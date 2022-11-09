package config

import (
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"k8s.io/utils/pointer"
	"reflect"
	"testing"
)

func TestValidateRoutes(t *testing.T) {
	tests := []struct {
		name          string
		overlaySubnet []string
		serviceSubnet []string
		wantErr       bool
	}{
		{
			name:          "overlaySubnet can't be empty",
			overlaySubnet: []string{},
			serviceSubnet: []string{"1.1.1.0/24"},
			wantErr:       true,
		}, {
			name:          "serviceSubnet can't be empty",
			overlaySubnet: []string{"1.1.1.0/24"},
			serviceSubnet: []string{},
			wantErr:       true,
		}, {
			name:          "ignore leading or trailing spaces",
			overlaySubnet: []string{" 1.1.1.0/24"},
			serviceSubnet: []string{" 2.2.2.0/24 "},
			wantErr:       false,
		}, {
			name:          "invalid cidr return err",
			overlaySubnet: []string{"abcd"},
			serviceSubnet: []string{"abcd"},
			wantErr:       true,
		}, {
			name:          "correct cidr config",
			overlaySubnet: []string{"10.69.0.0/12", "fd00:10:244::/64"},
			serviceSubnet: []string{"10.244.0.0/12", "fd00:10:69::/64"},
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := ValidateRoutes(tt.overlaySubnet, tt.serviceSubnet); (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoutes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRPFilterConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *ty.RPFilter
		want   *ty.RPFilter
	}{
		{
			name:   "no rp_filter config",
			config: nil,
			want: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			},
		}, {
			name: "enable rp_filter but no value given, we give default value to it",
			config: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  nil,
			},
			want: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			},
		}, {
			name: "give value but disable rp_filter",
			config: &ty.RPFilter{
				Enable: nil,
				Value:  pointer.Int32(2),
			},
			want: &ty.RPFilter{
				Enable: nil,
				Value:  pointer.Int32(2),
			},
		}, {
			name: "value must be 0/1/2, if not we set it to 2",
			config: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(10),
			},
			want: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			},
		}, {
			name: "correct rp_filter config",
			config: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(1),
			},
			want: &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateRPFilterConfig(tt.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateRPFilterConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateMigrateRouteConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *ty.MigrateRoute
		want   *ty.MigrateRoute
	}{
		{
			name:   "no config but we give default value -1",
			config: nil,
			want:   (*ty.MigrateRoute)(pointer.Int32(-1)),
		}, {
			name:   "value must be in -1,0,2, if not we set it to -1",
			config: (*ty.MigrateRoute)(pointer.Int32(1000)),
			want:   (*ty.MigrateRoute)(pointer.Int32(-1)),
		}, {
			name:   "correct config",
			config: (*ty.MigrateRoute)(pointer.Int32(-1)),
			want:   (*ty.MigrateRoute)(pointer.Int32(-1)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateMigrateRouteConfig(tt.config); *got != *tt.want {
				t.Errorf("ValidateMigrateRouteConfig() = %v, want %v", *got, *tt.want)
			}
		})
	}
}
