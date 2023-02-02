# Installation

*This guide shows how to install CNI-Meta-Plugins using [Helm](https://helm.sh/).*

## Generic

### helm
Set up the Helm repository.

```bash
helm repo add cni-meta-plugins https://spidernet-io.github.io/cni-plugins
```

Deploy CNI-Meta-Plugins using the default configuration options via Helm:

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

## Uninstall

Generally, you can uninstall CNI-META-PLUGINS release in this way:
### helm
```bash
helm uninstall meta-plugins -n kube-system
```

