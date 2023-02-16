# How to configuration

### A simple example

- **Veth**

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
                  }
              },{
                  "type": "veth",
                  "service_hijack_subnet": ["10.233.0.0/18"],
                  "overlay_hijack_subnet": ["10.244.0.0/16"],
                  "additional_hijack_subnet": [],
                  "migrate_route": -1,
                  "rp_filter": {
                      "set_host": true,
                      "value": 0
                  },
                  "skip_call": false
              }
          ]
      }
```

- **Router**

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
   name: macvlan-overlay
   namespace: kube-system
spec:
   config: |-
      {
          "cniVersion": "0.3.1",
          "name": "macvlan-overlay",
          "plugins": [
              {
                  "type": "macvlan",
                  "master": "eth0",
                  "mode": "bridge",
                  "ipam": {
                      "type": "spiderpool",
                  }
              },{
                  "type": "router",
                  "service_hijack_subnet": ["10.233.0.0/18"],
                  "overlay_hijack_subnet": ["10.244.0.0/16"],
                  "additional_hijack_subnet": [],
                  "migrate_route": -1,
                  "rp_filter": {
                      "set_host": true,
                      "value": 0
                  },
                  "skip_call": false
              }
          ]
      }
```

### In a dual-stack cluster, Configure IPv4/IPv6 subnet.

`service_hijack_subnet` and `overlay_hijack_subnet`: Supports configuration of multiple subnets, including ipv4/ipv6.

- **Veth**

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
                  }
              },{
                  "type": "veth",
                  "service_hijack_subnet": ["10.233.0.0/18","fd00:10:96::/112"],
                  "overlay_hijack_subnet": ["10.244.0.0/16","fd00:10:244::/112"],
                  "additional_hijack_subnet": [],
                  "migrate_route": -1,
                  "rp_filter": {
                      "set_host": true,
                      "value": 0
                  },
                  "skip_call": false
              }
          ]
      }
```

- **Router**

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
   name: macvlan-overlay
   namespace: kube-system
spec:
   config: |-
      {
          "cniVersion": "0.3.1",
          "name": "macvlan-overlay",
          "plugins": [
              {
                  "type": "macvlan",
                  "master": "eth0",
                  "mode": "bridge",
                  "ipam": {
                      "type": "spiderpool",
                  }
              },{
                  "type": "router",
                  "service_hijack_subnet": ["10.233.0.0/18","fd00:10:96::/112"],
                  "overlay_hijack_subnet": ["10.244.0.0/16","fd00:10:244::/112"],
                  "additional_hijack_subnet": [],
                  "migrate_route": -1,
                  "rp_filter": {
                      "set_host": true,
                      "value": 0
                  },
                  "skip_call": false
              }
          ]
      }
```

### Configure custom mac prefixes

`mac_preifx` is a unified mac address prefix, Length is 4 hex digits. Input format like: "1a:2b". If it's be empty, it's means disable this feature.

Take `veth` as an example:

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
                  }
              },{
                  "type": "veth",
                  "service_hijack_subnet": ["10.233.0.0/18","fd00:10:96::/112"],
                  "overlay_hijack_subnet": ["10.244.0.0/16","fd00:10:244::/112"],
                  "additional_hijack_subnet": [],
                  "migrate_route": -1,
                  "rp_filter": {
                      "set_host": true,
                      "value": 0
                  },
                  "mac_prefix": "0a:1b"
              }
          ]
      }
```

When pod is created, you see the mac address of pod(eth0) has been changed:

```shell
root@qf-master1:~# kubectl exec -it  macvlan-standalone-vlan0-f4d6d8776-9r9lf sh
kubectl exec [POD] [COMMAND] is DEPRECATED and will be removed in a future version. Use kubectl exec [POD] -- [COMMAND] instead.
/ # ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: tunl0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN qlen 1000
    link/ipip 0.0.0.0 brd 0.0.0.0
3: eth0@if7: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
    link/ether 0a:1b:0a:14:14:5f brd ff:ff:ff:ff:ff:ff
    inet 10.20.20.95/16 brd 10.20.255.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fd00:10:20::101/64 scope global
       valid_lft forever preferred_lft forever
    inet6 fe80::c94:54ff:fef9:d94a/64 scope link
       valid_lft forever preferred_lft forever
4: veth0@if513: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
    link/ether 86:4c:47:dc:11:dc brd ff:ff:ff:ff:ff:ff
    inet6 fe80::844c:47ff:fedc:11dc/64 scope link
       valid_lft forever preferred_lft forever
```

### Custom log options

`log_options` is used to config logger, as shown following:

```json
              "service_hijack_subnet": ["10.233.0.0/18","fd00:10:96::/112"],
              "overlay_hijack_subnet": ["10.244.0.0/16","fd00:10:244::/112"],
              "additional_hijack_subnet": [],
              "migrate_route": -1,
              "rp_filter": {
                "set_host": true,
                "value": 0
              },
              "log_options": {
                "log_level": "debug",
                "log_file": "/var/log/meta-plugins/router.log"
              },
```

You can config `log_level(default to debug)` and `log_file(default to/var/log/meta-plugins/router.log)`.
