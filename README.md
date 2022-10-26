# E2E

## Requirement

Run e2e-test on your local environment is required for following Cli tools, Please make sure these cli tools exists firstly: 

- *Kind*():
```shell
OS=$(uname | tr 'A-Z' 'a-z')
KIND_VERSION=v0.16.0
https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-${OS}-amd64
```
- *Helm*(v3.8+):
```shell
OS=$(uname | tr 'A-Z' 'a-z')
HELM_VERSION=v3.10.0
https://get.helm.sh/helm-${HELM_VERSION}-${OS}-amd64.tar.gz
```
- Docker()
- Ginkgo(2.1.3)
```shell
go install github.com/onsi/ginkgo/v2/ginkgo@v2.3.0 
```
- Kubectl()

## Setup kind cluster

### IPv4-only Cluster

```shell
make kind-init -e IP_FAMILY=ipv4
```

### IPv6-only Cluster

```shell
make kind-init -e IP_FAMILY=ipv6
```

### Dual-stack Cluster

```shell
make kind-init -e IP_FAMILY=dual
```

The default k8s version is v1.25.0, If you want to change it, You can use following command:

```shell
make kind-init -e K8S_VERSION=v1.26.0
```

NOTE:

> Before you can create ipv6-only or dual-stack cluster, you need to enable IPv6 support in the Docker daemon. Afterward, you can choose to use either IPv4 or IPv6 (or both) with any container, service, or network. Reference: [Docker: enable IPv6 support](https://docs.docker.com/config/daemon/ipv6/)

## Install

```shell
make install IP_FAMILY=dual(default) # or ipv4/ipv6
```

If you're running this in your local, Please using github proxy like the following:

```shell
make install RUN_ON_INSTALL=true
```

## Create vlan interface in kind-node

```shell
make vlan IP_FAMILY=dual
```

## Run e2e-test

```shell
make e2e 
```

## CleanUP

```shell
make clean 
```
