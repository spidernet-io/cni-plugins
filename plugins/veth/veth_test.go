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
)

var _ = Describe("Veth", func() {
	defer GinkgoRecover()
	Context("Test cmdDel", func() {
		It("test", func() {
			err := cmdDel(&skel.CmdArgs{})
			Expect(err).NotTo(HaveOccurred())
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
	})

	Context("Test setupVeth", func() {

		It("not first interface", func() {
			pr := &current.Result{}
			_, _, err := setupVeth(logger, testNetNs, true, containerID, pr)
			Expect(err).NotTo(HaveOccurred())
		})
		It("first interface", func() {
			pr := &current.Result{}
			_, _, err := setupVeth(logger, testNetNs, false, containerID, pr)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Test cmdAdd", func() {

		It("success", func() {
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
			args := &skel.CmdArgs{
				Netns:       testNetNs.Path(),
				ContainerID: containerID,
				StdinData:   stdin,
			}
			err := cmdAdd(args)
			Expect(err).NotTo(HaveOccurred())
		})

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

		It("GetRuleNumber < 0", func() {
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

})
