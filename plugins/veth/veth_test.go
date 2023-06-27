package main

import (
	"encoding/json"
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
	"github.com/vishvananda/netlink"
	"net"
)

var _ = Describe("Veth", func() {
	defer GinkgoRecover()
	Context("Test cmdDel", func() {
		It("test", func() {
			cmdDel(&skel.CmdArgs{})
			//Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Test cmdCheck", func() {
		It("test", func() {
			err := cmdCheck(&skel.CmdArgs{})
			Expect(err).To(Equal(errors.New("not implement it")))
		})
	})

	Context("Test parseConfig", func() {

		It("migrate_route no define, but we give it default value", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
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
			}`)
			_, err := parseConfig(stdin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("logOption not define,but we give it default value", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
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
			}`)
			_, err := parseConfig(stdin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("service or pod cidr must be define", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": [],
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
			}`)
			_, err := parseConfig(stdin)
			Expect(err).To(Equal(errors.New("the subnet of service clusterip must be given")))
		})

		It("json unmarshal err", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
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
			}`)
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(json.Unmarshal, errors.New("unmarshal err"))
			_, err := parseConfig(stdin)
			Expect(err).To(HaveOccurred())
		})

		It("parsePrevResult err", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
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
			}`)
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(version.ParsePrevResult, errors.New("parsePrevResult err"))
			_, err := parseConfig(stdin)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test setupVeth", func() {

		It("not first interface", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			pr := &current.Result{}
			patches.ApplyFuncReturn(ip.SetupVethWithName, hostInterface, conInterface, nil)
			_, _, err := setupVeth(logger, testNetNs, true, containerID, pr)
			Expect(err).NotTo(HaveOccurred())
		})
		It("first interface", func() {
			pr := &current.Result{}
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{netlink.LinkAttrs{HardwareAddr: net.HardwareAddr("test")}}, nil)
			_, _, err := setupVeth(logger, testNetNs, false, containerID, pr)
			Expect(err).NotTo(HaveOccurred())
		})

		It("first interface linkByName err", func() {
			pr := &current.Result{}
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{netlink.LinkAttrs{HardwareAddr: net.HardwareAddr("test")}}, errors.New("linkByName err"))
			_, _, err := setupVeth(logger, testNetNs, false, containerID, pr)
			Expect(err).To(HaveOccurred())
		})

		It(" SetupVethWithName err", func() {
			pr := &current.Result{}
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(ip.SetupVethWithName, nil, nil, errors.New("SetupVethWithName err"))
			_, _, err := setupVeth(logger, testNetNs, true, containerID, pr)
			Expect(err).To(HaveOccurred())
		})

		It(" setLinkup err", func() {
			pr := &current.Result{}
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(setLinkup, errors.New("setLinkup err"))
			_, _, err := setupVeth(logger, testNetNs, true, containerID, pr)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test cmdAdd", func() {

		//It("success", func() {
		//	var stdin = []byte(`{
		//		"cniVersion": "0.3.1",
		//		"name": "veth",
		//		"type": "veth",
		//		"service_hijack_subnet": ["10.244.64.0/18"],
		//		"overlay_hijack_subnet": ["10.244.0.0/18"],
		//		"rp_filter": {
		//			"enable": true,
		//			"value": 0
		//		},
		//		"prevResult": {
		//			"interfaces": [
		//				{"name": "net1"},
		//				{"name": "container", "sandbox":"netns"}
		//			],
		//			"ips": [
		//				{
		//					"version": "4",
		//					"address": "10.0.0.1/24",
		//					"gateway": "10.0.0.1",
		//					"interface": 0
		//				},
		//				{
		//					"version": "6",
		//					"address": "2001:db8:1::2/64",
		//					"gateway": "2001:db8:1::1",
		//					"interface": 0
		//				}
		//			]
		//		}
		//	}`)
		//	args := &skel.CmdArgs{
		//		Netns:       testNetNs.Path(),
		//		ContainerID: containerID,
		//		StdinData:   stdin,
		//	}
		//	err := cmdAdd(args)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		It("parse config err", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patches := gomonkey.NewPatches()
			patches.ApplyFuncReturn(parseConfig, nil, errors.New("parse config failed"))
			defer patches.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("config skip", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
	          "skip_call": true,
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).NotTo(HaveOccurred())
		})

		It("logging set failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patches := gomonkey.NewPatches()
			patches.ApplyFuncReturn(logging.SetLogOptions, errors.New("setting logger err"))
			defer patches.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("load k8s args failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patches := gomonkey.NewPatches()
			patches.ApplyFuncReturn(types.LoadArgs, errors.New("load k8s args err"))
			defer patches.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("prevResult parse failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(current.GetResult, nil, errors.New("parse prevResult failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("no prevResult", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				}
			}`)
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(Equal(errors.New("failed to find PrevResult, must be called as chained plugin")))
		})

		It("prevResult no ips", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
						{"name": "container", "sandbox":"netns"}
					]
				}
			}`)
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("prevResult no interface", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [],
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
			}`)
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(Equal(errors.New("failed to find interface from prevResult")))
		})

		It("prevResult no interface name", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name1": "net1"},
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
			}`)
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(Equal(errors.New("failed to find interface name from prevResult")))
		})

		It("get ns failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(ns.GetNS, nil, errors.New("get ns failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("check interface miss failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(utils.CheckInterfaceMiss, nil, errors.New("check interface miss failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("setupVeth failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(setupVeth, nil, nil, errors.New("setupVeth failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})
		//
		//It("GetRuleNumber < 0", func() {
		//	var stdin = []byte(`{
		//		"cniVersion": "0.3.1",
		//		"name": "veth",
		//		"type": "veth",
		//		"service_hijack_subnet": ["10.244.64.0/18"],
		//		"overlay_hijack_subnet": ["10.244.0.0/18"],
		//		"rp_filter": {
		//			"enable": true,
		//			"value": 0
		//		},
		//		"prevResult": {
		//			"interfaces": [
		//				{"name": "net1"},
		//				{"name": "container", "sandbox":"netns"}
		//			],
		//			"ips": [
		//				{
		//					"version": "4",
		//					"address": "10.0.0.1/24",
		//					"gateway": "10.0.0.1",
		//					"interface": 0
		//				},
		//				{
		//					"version": "6",
		//					"address": "2001:db8:1::2/64",
		//					"gateway": "2001:db8:1::1",
		//					"interface": 0
		//				}
		//			]
		//		}
		//	}`)
		//	patch := gomonkey.ApplyFuncReturn(utils.GetRuleNumber, -1)
		//	defer patch.Reset()
		//	args := &skel.CmdArgs{
		//		Netns:       testNetNs.Path(),
		//		ContainerID: containerID,
		//		StdinData:   stdin,
		//	}
		//	err := cmdAdd(args)
		//	Expect(err).To(HaveOccurred())
		//})

		It("GetHostIps failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(utils.GetHostIps, nil, errors.New("get GetHostIps failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("setupNeighborhood failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(setupNeighborhood, errors.New("setupNeighborhood failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("setupRoutes failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(setupRoutes, errors.New("setupRoutes failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("SysctlRPFilter failed", func() {
			var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
				"rp_filter": {
					"enable": true,
					"value": 0
				},
				"prevResult": {
					"interfaces": [
						{"name": "net1"},
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
			}`)
			patch := gomonkey.ApplyFuncReturn(utils.SysctlRPFilter, errors.New("SysctlRPFilter failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Test addSubnetRoute", func() {
		var routers = []string{"10.244.64.0/18", "2001:db8:1::2/64"}
		var destIPv4 = net.ParseIP("10.244.64.10")
		var destIPv6 = net.ParseIP("2001:db8:1::10")

		It("success", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.RouteAdd, nil)
			err := addSubnetRoute(logger, routers, 100, 1, true, true, &destIPv4, &destIPv6)
			Expect(err).NotTo(HaveOccurred())
		})

		It("parseCIDR err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(net.ParseCIDR, nil, nil, errors.New("parseCIDR err"))
			err := addSubnetRoute(logger, routers, 100, 1, true, true, &destIPv4, &destIPv6)
			Expect(err).To(HaveOccurred())
		})

		It("not gateway", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			err := addSubnetRoute(logger, routers, 100, 1, false, false, nil, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("add route err", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(netlink.RouteAdd, errors.New("add route err"))
			err := addSubnetRoute(logger, routers, 100, 1, true, true, &destIPv4, &destIPv6)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test setupRoutes", func() {

		var hInterface = &current.Interface{Name: hostVethName}
		var cInterface = &current.Interface{Name: conVethName}
		hostIPs := []string{"10.244.64.0/18"}
		conIPs := []string{"10.244.64.0/18"}
		var stdin = []byte(`{
				"cniVersion": "0.3.1",
				"name": "veth",
				"type": "veth",
				"service_hijack_subnet": ["10.244.64.0/18"],
				"overlay_hijack_subnet": ["10.244.0.0/18"],
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
			}`)
		conf, err := parseConfig(stdin)
		Expect(err).NotTo(HaveOccurred())
		It("success", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(utils.RouteAdd, nil, nil, nil)
			patches.ApplyFuncReturn(addSubnetRoute, nil)
			patches.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{netlink.LinkAttrs{HardwareAddr: net.HardwareAddr("test")}}, nil)
			err := setupRoutes(logger, testNetNs, 100, hInterface, cInterface, hostIPs, conIPs, conf, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

	})

})
