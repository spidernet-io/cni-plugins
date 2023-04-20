# CNI-Meta-Plugins

[![Run E2E Kind Test](https://github.com/spidernet-io/cni-plugins/actions/workflows/e2e-test.yaml/badge.svg)](https://github.com/spidernet-io/cni-plugins/actions/workflows/e2e-test.yaml)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cyclinder/92ef1f04e61af8f8b970c0b15f51c7a8/raw/comment.json)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cyclinder/6b05882662346c2592a432226bf3d249/raw/code-lines.json)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cyclinder/82aa5e4487e1870aa206c1d713429345/raw/todo.json)
[![codecov](https://codecov.io/gh/spidernet-io/cni-plugins/branch/main/graph/badge.svg?token=LcBT6jbJIT)](https://codecov.io/gh/spidernet-io/cni-plugins)
[![Auto Golang Lint And Unitest](https://github.com/spidernet-io/cni-plugins/actions/workflows/lint-golang.yaml/badge.svg)](https://github.com/spidernet-io/cni-plugins/actions/workflows/lint-golang.yaml)

## Status

Currently, the CNI-Meta-Plugins is under beta stage, not ready for production environment yet.

## Introduction

The CNI-Meta-Plugin is a collection of cni plugins designed to solve various network communication problems in the case of multiple network interface in the pod. Currently, it includes two plugins, router and veth

`veth`：Solve the communication problem when the default cni is macvlan or sriov by creating a veth pair. Refer to [Design](docs/develop/Design.md).

`router`: Solve the communication problem when the default cni is calico or cilium and the macvlan or sriov is configured with multiple network interfaces, move the routing table without cni to a new routing table. Refer to [Design](docs/develop/Design.md).

## Installation

Refer to [installation](./docs/usage/install.md).

## Quick Start

Refer to [demo](./docs/usage/basic.md).

## Features

- Supports creating multiple Macvlan NICs or SR-IOV NICs in Pods and solving network communication problems with multiple NICs via policy route.

    With `Multus`, You can create multiple NICs inside pod simply. But this may have some communication problems: Pod access failure due to default route. Let's see how the plugin works:

    Note: *You can invoke `veth` repeatedly, but not both `veth` and `router` plugins*

       `veth`: Keep the route of the first NIC(eth0) in the main table and move the routes of other NICs to the policy route.

       `router`: Keep the route of the latest NIC in the main table and move the routes of other NICs to the policy route.

    Examples refer to [two-macvlan-standalone](../example/macvlan/two-macvlan-standalone) and  [two-macvlan-overlay](../example/macvlan/two-macvlan-overlay)

- Supports Pod access to ClusterIP when macvlan is the default CNI

    As we all know, when the first NIC of Pod is created by `macvlan`, Pod can't access ClusterIP, we fix this problem by `veth` plugin.

- Fixed mac address support

    `veth` and `router` plugins supports fixed mac address. Including fixing the mac address of the NIC created by calico, macvlan, sriov.

       > Note: Can't change the mac address of the NIC created by cilium, Otherwise there are communication issues.

    Here is a cni configuration example:

    ```yaml
    apiVersion: k8s.cni.cncf.io/v1
    kind: NetworkAttachmentDefinition
    metadata:
      name: macvlan-standalone
      namespace: kube-system
    spec:
      config: |-
        {
            "cniVersion": "0.3.1",
            "name": "macvlan-standalone",
            "plugins": [
                {
                    "type": "macvlan",
                    "master": "eth0",
                    "mode": "bridge",
                    "ipam": {
                        "type": "spiderpool",
                        "log_level": "DEBUG",
                        "log_file_path": "/var/log/spidernet/spiderpool.log",
                        "log_file_max_size": 100,
                        "log_file_max_age": 30,
                        "log_file_max_count": 10
                    }
                },{
                    "type": "veth",
                    "service_hijack_subnet": ["10.233.0.0/18","fd00:10:96::/112"],
                    "overlay_hijack_subnet": [""10.244.0.0/16","fd00:10:244::/56"],
                    "additional_hijack_subnet": [],
                    "migrate_route": -1,
                    "rp_filter": {
                        "set_host": true,
                        "value": 0
                    },
                    "skip_call": false,
                    "mac_prefix": "0a:1b"
                }
            ]
        }
    ```

    `mac_prefix` an is the unified mac address prefix, Length is 4 hex digits. Input format like: "1a:2b".
- Detect IPv4/IPv6 address conflicting

The veth and router plugins support IPv4 and IPv6 addresses conflict detection using arp or ndp protocols for the pod's IP, and return an error if the IP is found to be already in use by another host on the LAN.
You can enable this feature in the following way:

```json
             "log_options": {
                  "log_level": "debug",
                  "log_file": "/var/log/meta-plugins/router.log"
             },
             "ip_conflict": {
                  "enabled": true,
                  "interval": "1s",
                  "retries": 5
             },
```

more details refer to [Configuration](docs/usage/config.md)

# License

CNI-Meta-Plugin is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
