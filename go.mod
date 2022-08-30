module cni-plugins

go 1.19

require (
	github.com/containernetworking/cni v1.1.2
	github.com/containernetworking/plugins v1.1.1
	github.com/spidernet-io/veth-plugin v0.1.2
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	k8s.io/utils v0.0.0-20220823124924-e9cbc92d1a73
)

require (
	github.com/coreos/go-iptables v0.6.0 // indirect
	github.com/safchain/ethtool v0.0.0-20210803160452-9aa261dae9b1 // indirect
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f // indirect
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
