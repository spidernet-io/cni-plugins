# CNI-Meta-Plugin

- **veth**
- **router**

## Why

## How to start

### Examples

Multus Net-Atta-Def CRD examples:

- Macvlan-standalone

```shell
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: macvlan-standalone
  namespace: kube-system
spec:
  config: |-
    {
        "cniVersion": "0.3.0",
        "name": "macvlan-standalone",
        "plugins": [
            {
                "type": "macvlan",
                "master": "ens192",  # macvlan master insterface
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
                "routes": [{"dst": "10.244.0.0/18"},{"dst": "10.244.64.0/18"}], # calico/service subnet
                "rp_filter": {
                    "enable": true,
                    "value": 2
                },
                "skip_call": false
            }
        ]
    }

```

- Macvlan-overlay

Calico + Macvlan:

```shell
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: macvlan-overlay
  namespace: kube-system
spec:
  config: |-
    {
        "cniVersion": "0.3.0",
        "name": "macvlan-overlay",
        "plugins": [
            {
                "type": "macvlan",
                "master": "ens192",
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
                "type": "router",
                "routes": [{"dst": "10.244.0.0/18"},{"dst": "10.244.64.0/18"}], # service/calico subnet
                "rp_filter": {
                    "enable": true,
                    "value": 2
                },
                "delDefaultRoute4": true,
                "delDefaultRoute6": true,
                "defaultOverlayInterface": eth0,
                "skip_call": false
            }
        ]
    }

```

## Notice 
