# Quick start

*Let's start some Pods with CNI-Meta-Plugins in approximately 5 minutes.*

## Prepare k8s environment

Prepare a k8s environment with the default cni as calico. You can use kind or kubeadm to build the cluster

## Install CNI-Meta-Plugins

### helm
1. Set up the Helm repository.

    ```bash
    helm repo add cni-meta-plugins https://spidernet-io.github.io/cni-plugins
    ```

2. Deploy CNI-Meta-Plugins with the following command.

   ```bash
   helm install meta-plugins cni-meta-plugins/meta-plugins -n kube-system 
   ```

More details about [CNI-META-PLUGINS charts parameters](https://github.com/spidernet-io/cni-plugins/blob/main/charts/meta-plugins/README.md).

>After installation, you can see the router and veth binaries in the/opt/cni/bin directory of each node.

### binary
If you don't want to use helm for installation, you can download the binary file directly.
```bash
# You need to download and decompress at each node
wget https://github.com/spidernet-io/cni-plugins/releases/download/v0.2.1/spider-cni-plugins-linux-amd64-v0.2.1.tar
tar xvfzp /root/spider-cni-plugins-linux-amd64-v0.2.1.tar -C /opt/cni/bin
```

## Install multus-cni

We use multius-cni to create a container network with multiple network interfaces
Deploy multus-cni with the following command.
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml
   ```

## Create multus-cni CR

This yaml used in overlay mode

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
                      "log_level": "DEBUG",
                      "log_file_path": "/var/log/spidernet/spiderpool.log",
                      "log_file_max_size": 100,
                      "log_file_max_age": 30,
                      "log_file_max_count": 10
                  }
              },{
                  "type": "router",
                  "service_hijack_subnet": ["10.233.0.0/18","fd00:10:96::/112"],
                  "overlay_hijack_subnet": ["10.244.0.0/16","fd00:10:244::/56"],
                  "additional_hijack_subnet": [],
                  "migrate_route": -1,
                  "rp_filter": {
                      "set_host": true,
                      "value": 0
                  },
                  "overlay_interface": "eth0",
                  "skip_call": false
              }
          ]
      }
```

This yaml used in underlay mode

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
                  "overlay_hijack_subnet": ["10.244.0.0/16","fd00:10:244::/56"],
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

Use `kubectl apply -f` command create above two resources

>If you want to use another ipam, please configure the ipam field by yourself.
>if you want to use spiderpool as ipam, you need install spiderpool refer to [install spiderpool](https://github.com/spidernet-io/spiderpool/blob/main/docs/usage/install.md)

## Run

#### Underlay mode

Create a deployment using the following yaml 

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
   name: macvlan-standalone-deploy
spec:
   replicas: 3
   selector:
      matchLabels:
         app: macvlan-standalone-deploy
   template:
      metadata:
         annotations:
            v1.multus-cni.io/default-network: kube-system/macvlan-standalone
         labels:
            app: macvlan-standalone-deploy
      spec:
         containers:
            - name: macvlan-standalone-deploy
              image: busybox
              imagePullPolicy: IfNotPresent
              command: ["/bin/sh", "-c", "trap : TERM INT; sleep infinity & wait"]
```
Use this command `kubectl apply -f` create deployment.

>Multius uses `annotations` to configure the network, so we need to add the `v1.multus-cni.io/default-network: kube-system/macvlan-standalone` field to the annotations of deployment, which means using the `macvlan-standalone` under the `kube-system` namespace as the cni configuration to create the network.
>Please refer to the [multus](https://github.com/k8snetworkplumbingwg/multus-cni)

You will see three pods with ip assigned by macvlan:

```bash
kubectl get pod -o wide
NAME                               READY   STATUS    RESTARTS   AGE   IP              NODE                 NOMINATED NODE   READINESS GATES
macvlan-standalone-deploy-86cf469554-2c9jg   1/1     Running   0          10s   172.18.24.124   dual-worker          <none>           <none>
macvlan-standalone-deploy-86cf469554-fsq87   1/1     Running   0          10s   172.18.73.64    dual-control-plane   <none>           <none>
macvlan-standalone-deploy-86cf469554-sgdl8   1/1     Running   0          10s   172.18.38.218   dual-worker          <none>           <none>
```

Let's see inside the pod:

```shell
kubectl exec -it macvlan-standalone-deploy-86cf469554-2c9jg sh
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
      inet 172.18.24.124/16 brd 10.20.255.255 scope global eth0
      valid_lft forever preferred_lft forever
      inet6 fd00:172:18::101/64 scope global
      valid_lft forever preferred_lft forever
      inet6 fe80::c94:54ff:fef9:d94a/64 scope link
      valid_lft forever preferred_lft forever
4: veth0@if513: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
      link/ether 86:4c:47:dc:11:dc brd ff:ff:ff:ff:ff:ff
      inet6 fe80::844c:47ff:fedc:11dc/64 scope link
      valid_lft forever preferred_lft forever
/ # ip r
default via 172.18.0.1 dev eth0   # the default gw of underlay network
10.244.64.0/18 via 172.18.0.2 dev veth0 # to <clusterIPs> via veth0
172.18.0.0/16 dev eth0 scope link  src 172.18.24.124 
10.244.0.0/18 via 172.18.0.2 dev veth0 # to calico pod via veth0
/ # # ping test for gateway
/ # ping 172.18.0.1
PING 172.18.0.1 (172.18.0.1) 56(84) bytes of data.
64 bytes from 172.18.0.1: icmp_seq=1 ttl=255 time=0.492 ms
64 bytes from 172.18.0.1: icmp_seq=2 ttl=255 time=0.517 ms
64 bytes from 172.18.0.1: icmp_seq=3 ttl=255 time=0.464 ms
^C
--- 172.18.0.1 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2042ms
rtt min/avg/max/mdev = 0.464/0.491/0.517/0.021 ms
```

#### Overlay mode

Create a deployment using the following yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
   name: macvlan-overlay-deploy
spec:
   replicas: 3
   selector:
      matchLabels:
         app: macvlan-overlay-deploy
   template:
      metadata:
         annotations:
            k8s.v1.cni.cncf.io/networks: kube-system/macvlan-overlay
         labels:
            app: macvlan-overlay-deploy
      spec:
         containers:
            - name: macvlan-overlay-deploy
              image: busybox
              imagePullPolicy: IfNotPresent
              command: ["/bin/sh", "-c", "trap : TERM INT; sleep infinity & wait"]
```
Use this command `kubectl apply -f` create deployment.

>Multius uses `annotations` to configure the network, so we need to add the `k8s.v1.cni.cncf.io/networks: kube-system/macvlan-overlay` field to the annotations of deployment, which means using the `acvlan-overlay` under the `kube-system` namespace as the cni configuration to create the network.
>Please refer to the [multus](https://github.com/k8snetworkplumbingwg/multus-cni)

You will see three pods. There are two kinds of ip in the pod: calico and macvlan

```bash
kubectl get pod -o wide
NAME                              READY   STATUS    RESTARTS   AGE   IP               NODE                 NOMINATED NODE   READINESS GATES
macvlan-overlay-deploy-5657cb876b-8x8sb   1/1     Running   0          8s    10.244.181.69    dual-worker          <none>           <none>
macvlan-overlay-deploy-5657cb876b-9xxqp   1/1     Running   0          8s    10.244.104.200   dual-control-plane   <none>           <none>
macvlan-overlay-deploy-5657cb876b-r88g4   1/1     Running   0          8s    10.244.181.70    dual-worker          <none>           <none>
```

Use the following command to view all the ip addresses of the pod.

```bash
   kubectl get pod macvlan-overlay-deploy-5657cb876b-8x8sb -o yaml
```

```yaml
annotations:
    cni.projectcalico.org/containerID: dcdf9b796514ebb70a848a8936040c1353debbbb7f86643a2c48965a42c84a58
    cni.projectcalico.org/podIP: 10.244.181.69/32
    cni.projectcalico.org/podIPs: 10.244.181.69/32,fd00:10:244:37:a5bd:7eaf:211e:b544/128
    ipam.spidernet.io/assigned-net1: '{"interface":"net1","ipv4pool":"default-v4-ippool","ipv6pool":"default-v6-ippool","ipv4":"172.18.87.211/16","ipv6":"fc00:f853:ccd:e793::888d/64","vlan":0}'
    k8s.v1.cni.cncf.io/network-status: |-
      [{
          "name": "kube-system/k8s-pod-network",
          "ips": [
              "10.244.181.69",
              "fd00:10:244:37:a5bd:7eaf:211e:b544"
          ],
          "default": true,
          "dns": {}
      },{
          "name": "kube-system/macvlan-overlay",
          "interface": "net1",
          "ips": [
              "172.18.87.211",
              "fc00:f853:ccd:e793::888d"
          ],
          "mac": "9a:a5:04:39:8d:54",
          "dns": {}
      }]
    k8s.v1.cni.cncf.io/networks: kube-system/macvlan-overlay
    k8s.v1.cni.cncf.io/networks-status: |-
      [{
          "name": "kube-system/k8s-pod-network",
          "ips": [
              "10.244.181.69",
              "fd00:10:244:37:a5bd:7eaf:211e:b544"
          ],
          "default": true,
          "dns": {}
      },{
          "name": "kube-system/macvlan-overlay",
          "interface": "net1",
          "ips": [
              "172.18.87.211",
              "fc00:f853:ccd:e793::888d"
          ],
          "mac": "9a:a5:04:39:8d:54",
          "dns": {}
      }]
```

Let's see inside the pod:

```shell
kubectl exec -it macvlan-overlay-deploy-5657cb876b-8x8sb sh
kubectl exec [POD] [COMMAND] is DEPRECATED and will be removed in a future version. Use kubectl exec [POD] -- [COMMAND] instead.
sh-4.4# ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: tunl0@NONE: <NOARP> mtu 1480 qdisc noop state DOWN group default qlen 1000
    link/ipip 0.0.0.0 brd 0.0.0.0
4: eth0@if3108: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UP group default
    link/ether 42:3a:7f:50:ef:29 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 10.244.181.69/32 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fe80::403a:7fff:fe50:ef29/64 scope link
       valid_lft forever preferred_lft forever
5: net1@if7: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 8a:9f:22:77:bd:cb brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.18.24.120/16 brd 10.20.255.255 scope global net1
       valid_lft forever preferred_lft forever
    inet6 fd00:10:20::aaa8/64 scope global
       valid_lft forever preferred_lft forever
    inet6 fe80::889f:22ff:fe77:bdcb/64 scope link
       valid_lft forever preferred_lft forever
```

There are two NICs inside the pod, The First one is created by `calico`, and second one is created by `macvlan`. Let's see route table:

```shell
sh-4.4# ip rule
0:	from all lookup local
32763:	from 10.244.181.69 lookup 100 # from eth0 lookup table 100
32764:	from all to 10.244.0.0/18 lookup 100 # to <cluster IPs> lookup table 100
32765:	from all to 10.244.64.0/18 lookup 100 # to <calico pods> lookup table 100
32766:	from all lookup main
32767:	from all lookup default
```

Show route tables for table 100:

```shell
sh-4.4# ip r show t 100
default via 169.254.1.1 dev eth0 # route tables of calico
169.254.1.1 dev eth0 scope link  # route tables of calico
172.18.0.0/16 dev net1 proto kernel scope link src 172.18.24.120
172.18.0.2 dev eth0 scope link
```

show route tables for table main:

```shell
sh-4.4# ip r
default via 172.18.0.1 dev net1 # default gateway of underlay network
172.18.0.0/16 dev net1 proto kernel scope link src 172.18.24.120
169.254.1.1 dev eth0 scope link
172.18.0.2 dev eth0 scope link
```

ping test for gateway:

```shell
sh-4.4# ping 172.18.0.1
PING 172.18.0.1 (172.18.0.1) 56(84) bytes of data.
64 bytes from 172.18.0.1: icmp_seq=1 ttl=255 time=0.492 ms
64 bytes from 172.18.0.1: icmp_seq=2 ttl=255 time=0.517 ms
64 bytes from 172.18.0.1: icmp_seq=3 ttl=255 time=0.464 ms
^C
--- 172.18.0.1 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2042ms
rtt min/avg/max/mdev = 0.464/0.491/0.517/0.021 ms
```