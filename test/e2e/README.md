# e2e case

e2e-kind 环境安装起来后, 除了会安装 Multus、Spiderpool等组件之后，还会创建 Multus CRD实例及Spiderpool池，以便测试:

Multus CRD:

```shell
-  macvlan-standalone-vlan0
-  macvlan-standalone-vlan100
-  macvlan-overlay-vlan0
-  macvlan-overlay-vlan100
```

Spiderpool:

```shell
- default-v4-ippool: ipv4缺省池
- default-v6-ippool: ipv6缺省池
- vlan100-v4
- vlan100-v6
```


## case1: macvlan-standalone-one

`macvlan-standalone-one`: 表示在 Pod 中只插入一张由 macvlan 分配的网卡, 通过在Pod 的 annotations 中插入以下的注解实现:

```shell
v1.multus-cni.io/default-network: kube-system/macvlan-standalone-vlan0
```

## case2: macvlan-standalone-two

`macvlan-standalone-two`: 表示在 Pod 中插入两张由 macvlan 分配的网卡, 通过在Pod 的 annotations 中插入以下的注解实现:

```shell
v1.multus-cni.io/default-network: kube-system/macvlan-standalone-vlan0
k8s.v1.cni.cncf.io/networks:  kube-system/macvlan-standalone-vlan100
```

## case3: macvlan-overlay-one

`macvlan-overlay-one`: 表示在 Pod 中插入两张网卡, 第一张有缺省CNI(calico)分配, 第二张由 macvlan 分配, 通过在Pod 的 annotations 中插入以下的注解实现:

```shell
k8s.v1.cni.cncf.io/networks:  kube-system/macvlan-overlay-vlan100
```

## case4: macvlan-overlay-two

`macvlan-overlay-two`: 表示在 Pod 中插入两张Macvlan网卡, 第一张有缺省CNI(calico)分配, 第二、三张由 macvlan 分配, 通过在Pod 的 annotations 中插入以下的注解实现:

```shell
k8s.v1.cni.cncf.io/networks:  kube-system/macvlan-overlay-vlan0,kube-system/macvlan-overlay-vlan100
```

## 测试主要内容

- Pod之间的联通性,主要是跨节点通讯, 不同网卡要求都能够联通。
- Pod与主机之间的联通性, 不同网卡都要求联通。
- Pod访问 ClusterIP
- 主机访问ClusterIP
- 主机访问 NodePort
- 集群外访问 Pod
