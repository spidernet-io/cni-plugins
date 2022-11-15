package main

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	It("test parse config", func() {
		conf := `{
		"cniVersion": "0.3.1",
		"name": "router",
		"type": "router",
		"service_hijack_subnet": ["10.244.64.0/18"],
		"overlay_hijack_subnet": ["10.244.0.0/18"],
		"rp_filter": {
			"enable": true,
			"value": 0
		},
		"overlay_interface": "eth0",
		"migrate_route": -1,
		"log_options": {
			"log_level": "debug"
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
		err := json.Unmarshal([]byte(conf), pluginConf)
		Expect(err).NotTo(HaveOccurred())
	})

})
