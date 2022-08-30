package main

import (
	"encoding/json"
	"testing"
)

func Test_parseConfig(t *testing.T) {
	conf := `{
		"cniVersion": "0.3.1",
		"name": "router",
		"type": "router",
		"routes": [
			{"gw": 172.17.8.1"},
	   		{"dst": "172.16.0.0/24"}
		],
		"rp_filter": {
			"enable": true,
			"value": 0
		},
		"prevResult": {
			"interfaces": [
				{"name": "host"},
				{"name": "container", "sandbox":"netns"}
			],
			"ips": [
				{
					"version": "4",
					"address": "10.0.0.1/24",
					"gateway": "10.0.0.1",
					"interface": 0
				},
				{
					"version": "6",
					"address": "2001:db8:1::2/64",
					"gateway": "2001:db8:1::1",
					"interface": 0
				}
			]
		}
	}`
	pluginConf := &PluginConf{}
	if err := json.Unmarshal([]byte(conf), pluginConf); err != nil {
		t.Fatal(err)
	}

	t.Log(pluginConf.RPFilter)

}
