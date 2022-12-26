package main

import (
	"errors"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/logging"
	"github.com/spidernet-io/cni-plugins/pkg/utils"
	"github.com/vishvananda/netlink"
	"net"
)

var _ = Describe("Router", func() {

	Context("Test parse config", func() {
		It("success", func() {
			var stdin = []byte(`{
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
	}`)
			_, err := parseConfig(stdin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("success not default interface ", func() {
			var stdin = []byte(`{
		"cniVersion": "0.3.1",
		"name": "router",
		"type": "router",
		"service_hijack_subnet": ["10.244.64.0/18"],
		"overlay_hijack_subnet": ["10.244.0.0/18"],
		"rp_filter": {
			"enable": true,
			"value": 0
		},
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
	}`)
			_, err := parseConfig(stdin)
			Expect(err).NotTo(HaveOccurred())
		})

		It("parse failed", func() {
			var stdin = []byte(`{
		"cniVersion": "0.3.1",
		"name": "router",
		"type": "router",
		"service_hijack_subnet": ["10.244.64.0/18"],
		"overlay_hijack_subnet": ["10.244.0.0/18"],
		"rp_filter": {
			"enable": true,
			"value": 0,

		},,
		"overlay_interface": "eth0",
		"migrate_route": -1,,,
		"log_options": {
			"log_level": "debug",
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

			Expect(err).To(HaveOccurred())
		})

		It("version failed", func() {
			var stdin = []byte(`{
		"cniVersion": "0.9.1",
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
	}`)
			_, err := parseConfig(stdin)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test addHostIPRoute", func() {
		It("underlay", func() {
			err := addHostIPRoute(logger, testNetNs, 101, secondifName, false, true, true)
			Expect(err).NotTo(HaveOccurred())
		})
		It("overlay", func() {
			err := addHostIPRoute(logger, testNetNs, 100, secondifName, false, true, true)
			Expect(err).NotTo(HaveOccurred())
		})
		It("when main cni is sroiv, don't need to add route", func() {
			err := addHostIPRoute(logger, testNetNs, 100, secondifName, true, true, true)
			Expect(err).NotTo(HaveOccurred())
		})

		It("GetHostIps failed", func() {
			patchees := gomonkey.NewPatches()
			defer patchees.Reset()
			patchees.ApplyFuncReturn(utils.GetHostIps, nil, errors.New("GetHostIps err"))
			err := addHostIPRoute(logger, testNetNs, 101, secondifName, false, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("overlay add route failed", func() {
			patchees := gomonkey.NewPatches()
			defer patchees.Reset()
			patchees.ApplyFuncReturn(utils.RouteAdd, nil, nil, errors.New("utils.RouteAdd err"))
			err := addHostIPRoute(logger, testNetNs, 100, secondifName, false, true, true)
			Expect(err).To(HaveOccurred())
		})

		It("underlay add route failed failed", func() {
			patchees := gomonkey.NewPatches()
			defer patchees.Reset()
			patchees.ApplyFuncReturn(utils.RouteAdd, nil, nil, errors.New("RouteAdd err"))
			err := addHostIPRoute(logger, testNetNs, 101, secondifName, false, true, true)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test addChainedIPRoute", func() {

		It("success", func() {
			patch := gomonkey.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, nil)
			defer patch.Reset()
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, overlayifName, defaultInterfaceIPs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("get host ip failed", func() {
			patches := gomonkey.NewPatches()
			patches.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, errors.New("get host ip failed"))
			defer patches.Reset()
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})

		It("failed to get netlink", func() {
			patches := gomonkey.NewPatches()
			patches.ApplyFuncReturn(netlink.LinkByName, nil, errors.New("link no found"))
			defer patches.Reset()
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})

		It("skip call addChainedIPRoute", func() {
			patches := gomonkey.NewPatches()
			patches.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, nil)
			defer patches.Reset()
			err := addChainedIPRoute(logger, testNetNs, true, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("netlink.LinkByIndex failed", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, nil)
			patches.ApplyFuncReturn(netlink.LinkByIndex, nil, errors.New("netlink.LinkByIndex err"))
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})

		It("parseCIDR failed", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, nil)
			patches.ApplyFuncReturn(net.ParseCIDR, nil, nil, errors.New("parseCIDR err"))
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})

		It("parseCIDR failed", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, nil)
			patches.ApplyFuncSeq(net.ParseCIDR, []gomonkey.OutputCell{
				{Values: gomonkey.Params{nil, nil, nil}},
				{Values: gomonkey.Params{nil, nil, errors.New("parseCIDR err")}},
			})
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})

		It("netlink.RuleAdd failed", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFuncReturn(utils.GetHostIps, []string{"10.96.0.13/16"}, nil)
			patches.ApplyFuncReturn(netlink.RuleAdd, errors.New("netlink.RuleAdd err"))
			err := addChainedIPRoute(logger, testNetNs, false, true, true, 100, secondifName, defaultInterfaceIPs)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test cmdAdd", func() {

		It("success", func() {
			var stdin = []byte(`{
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

		It("parse config failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(parseConfig, nil, errors.New("parse config failed"))
			defer patch.Reset()
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
		"name": "router",
		"type": "router",
		"service_hijack_subnet": ["10.244.64.0/18"],
		"overlay_hijack_subnet": ["10.244.0.0/18"],
		"rp_filter": {
			"enable": true,
			"value": 0
		},
		"overlay_interface": "eth0",
        "skip_call": true,
		"migrate_route": -1,
		"log_options": {
			"log_level": "debug"
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
			patch := gomonkey.ApplyFuncReturn(logging.SetLogOptions, errors.New("setting logger err"))
			defer patch.Reset()
			var stdin = []byte(`{
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
			Expect(err).To(HaveOccurred())
		})

		It("load k8s args failed", func() {
			patch := gomonkey.ApplyFuncReturn(types.LoadArgs, errors.New("load k8s args err"))
			defer patch.Reset()
			var stdin = []byte(`{
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
			Expect(err).To(HaveOccurred())
		})

		It("prevResult parse failed", func() {
			var stdin = []byte(`{
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
			"interfaces2": [
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
			Expect(err).To(Equal(errors.New("failed to find interface from prevResult")))
		})

		It("prevResult no interface name", func() {
			var stdin = []byte(`{
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

		It("get chained interface ip failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(utils.GetChainedInterfaceIps, nil, errors.New("get chained interface ip failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("get GetRuleNumber < 1 ", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(utils.GetRuleNumber, -1)
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("AddStaticNeighTable failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(utils.AddStaticNeighTable, errors.New("AddStaticNeighTable failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("EnableIpv6Sysctl failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(utils.EnableIpv6Sysctl, errors.New("EnableIpv6Sysctl failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("addChainedIPRoute failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(addChainedIPRoute, errors.New("addChainedIPRoute failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("addHostIPRoute failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(addHostIPRoute, errors.New("addHostIPRoute failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("HijackCustomSubnet failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(utils.HijackCustomSubnet, errors.New("HijackCustomSubnet failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).To(HaveOccurred())
		})

		It("MigrateRoute failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(utils.MigrateRoute, errors.New("MigrateRoute failed"))
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

	Context("Test cmdDel", func() {

		It("success", func() {
			var stdin = []byte(`{
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
			err := cmdDel(args)
			Expect(err).NotTo(HaveOccurred())
		})

		It("parse config failed", func() {
			var stdin = []byte(`{
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
			patch := gomonkey.ApplyFuncReturn(parseConfig, nil, errors.New("parse config failed"))
			defer patch.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdDel(args)
			Expect(err).To(HaveOccurred())
		})

		It("logging set failed", func() {
			patch := gomonkey.ApplyFuncReturn(logging.SetLogOptions, errors.New("setting logger err"))
			defer patch.Reset()
			var stdin = []byte(`{
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
			err := cmdDel(args)
			Expect(err).To(HaveOccurred())
		})

		It("load k8s args failed", func() {
			patch := gomonkey.ApplyFuncReturn(types.LoadArgs, errors.New("load k8s args err"))
			defer patch.Reset()
			var stdin = []byte(`{
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
			err := cmdDel(args)
			Expect(err).To(HaveOccurred())
		})

		It("get ns failed", func() {
			var stdin = []byte(`{
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
			err := cmdDel(args)
			Expect(err).To(HaveOccurred())
		})

		It("get chained ips failed", func() {
			var stdin = []byte(`{
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
			patches.ApplyFuncReturn(utils.GetChainedInterfaceIps, []string{"10.96.0.13/16"}, nil)
			defer patches.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdDel(args)
			Expect(err).NotTo(HaveOccurred())
		})

		It("del rule failed", func() {
			var stdin = []byte(`{
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
			patches.ApplyFuncReturn(utils.GetChainedInterfaceIps, []string{"10.96.0.13/16"}, nil)
			patches.ApplyFuncReturn(utils.RuleDel, errors.New("rule del failed"))
			defer patches.Reset()
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdDel(args)
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Test cmdCheck", func() {

		It("success", func() {
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
			}
			err := cmdCheck(args)
			Expect(err).To(Equal(errors.New("not implement it")))
		})
	})
})
