package main

import (
	"net"
	"reflect"
	"testing"
)

func Test_filterIPs(t *testing.T) {

	tests := []struct {
		name string
		ips  []string
		want []string
	}{
		{
			name: "only one ipv4 ip",
			ips:  []string{"192.168.1.1"},
			want: []string{"192.168.1.1"},
		}, {
			name: "ipv4 and ipv6 ip",
			ips:  []string{"192.168.1.1", "fd00:1033::f197:b232:eaa:bac0"},
			want: []string{"192.168.1.1", "fd00:1033::f197:b232:eaa:bac0"},
		}, {
			name: "more ipv4 and ipv6 ip",
			ips:  []string{"192.168.1.1", "fd00:1033::f197:b232:eaa:bac0", "192.168.2.1", "fd00:1033::172:17:8:120"},
			want: []string{"192.168.1.1", "fd00:1033::f197:b232:eaa:bac0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv4, ipv6 := false, false
			got := []string{}
			for _, ip := range tt.ips {
				netIP := net.ParseIP(ip)
				ipv4, ipv6, got = veth_plugin.filterIPs(netIP, ipv4, ipv6, got)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterIPs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
