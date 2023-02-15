# Features

## Supports creating multiple Macvlan NICs or SR-IOV NICs in Pods and solving network communication problems with multiple NICs via policy route.

With `Multus`, You can create multiple NICs inside pod simply. But this may have some communication problems: Pod access failure due to default route. Let's see how the plugin works:

Note: *You can invoke `veth` repeatedly, but not both `veth` and `router` plugins*

`veth`: Keep the route of the first NIC(eth0) in the main table and move the routes of other NICs to the policy route.
`router`: Keep the route of the latest NIC in the main table and move the routes of other NICs to the policy route.

Examples refer to [two-macvlan-standalone](../example/macvlan/two-macvlan-standalone) and  [two-macvlan-overlay](../example/macvlan/two-macvlan-overlay)

## Fixed mac address support

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

`mac_prefix` a is the unified mac address prefix, Length is 4 hex digits. Input format like: "1a:2b".
