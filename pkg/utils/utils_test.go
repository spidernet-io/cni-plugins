package utils

import "testing"

func Test_GetRuleNumber(t *testing.T) {

	tests := []struct {
		name   string
		iface  string
		number int
	}{
		{
			"first macvlan interface",
			"net1",
			100,
		}, {
			"second macvlan interface",
			"net2",
			101,
		}, {
			"invalid interface name",
			"eth0",
			-1,
		}, {
			"invalid interface name",
			"ens192",
			-1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRuleNumber(tt.iface); got != tt.number {
				t.Errorf("getRuleNumber() = %v, want %v", got, tt.number)
			}
		})
	}
}
