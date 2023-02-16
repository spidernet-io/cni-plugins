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

## Features

Refer to [Features](./docs/description/feature.md)

## Installation

Refer to [installation](./docs/usage/install.md).

## Quick Start

Refer to [demo](./docs/usage/basic.md).

## Development

Refer to [features](docs/description/feature.md)

# License

CNI-Meta-Plugin is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
