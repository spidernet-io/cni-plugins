# Descriptions

This document introduces the compatibility of these underlay plugins(`veth` and `router`) with some Network addons.

## NETWORK ADDON COMPATIBILITY

|Network Addon|Compatible|
|----|----|
|Calico|Yes|
|Cilium|Mostly (see Known issues)|
|Macvlan|Yes|
|SR-IOV CNI|Yes|

## Known Issues

- In cilium-overlay mode(cilium + macvlan/sriov + router), The node cannot access the underlay IP of the Pod on that node. See [#Issue 143](https://github.com/spidernet-io/cni-plugins/issues/143)

> In a cilium cluster, Create a macvlan pod with router-plugins(demo see [macvlan-overlay](../example/macvlan/macvlan-overlay)), There are two NICs in the Pod, First one is created by `cilium`, Second one is created by `macvlan`. Because of the mechanism of macvlan bridge mode, The master interface of `macvlan` cannot communicate directly with the sub-interfaces, so we forward the traffic through the `veth` device of the cilium/calico pod. Calico is works but cilium not. cilium is not based on a legacy netfilter implementation. The ebpf program that cilium mounts to the lxc* device explicitly drops reply packets when they are sent from the pod, and the source IP of the reply packet is the underlay IP of the pod, so cilium think this to be invalid.

- In cilium + macvlan/sriov + veth mode, Clients outside the cluster cannot access the nodePort service. See [#Issue 142](https://github.com/spidernet-io/cni-plugins/issues/142)

> In a cilium cluster, Create a macvlan pod with veth-plugins(demo see [macvlan-standalone](../example/macvlan/macvlan-standalone)), There are only one NIC in the Pod, which is created by `macvlan`. And create a nodePort service, Clients outside the cluster cannot access it. At present, there is no definite conclusion on this issue. Guess it has something to do with `SNAT`. Cilium is not based on the traditional netfilter implementation.

## Time cost on plugins invoke(Time in milliseconds)

|Plugin|min|max|avg|
|----|----|----|----|
|veth|12.7|94.1|41.7|
|router|11.3|70.7|30.7|
