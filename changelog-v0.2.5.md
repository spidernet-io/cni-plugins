## What's Changed
* add zap logger and fix bug by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/30
* opt image build and adjust chart version by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/31
* adjust the struct of the routes by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/32
* support multi underlay-cni interface by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/33
* fix ci: wrong binary file path by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/36
* fix the issue of ipv6 communication failure between pods and host by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/44
* trivy scan by @weizhoublue in https://github.com/spidernet-io/cni-plugins/pull/53
* ignore error when del rule not exist by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/48
* add e2e kind by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/62
* add github action by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/65
* fix typo & bump to v0.1.8 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/66
* fix e2e load images err by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/67
* modify e2e kind-init by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/70
* fix ci by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/78
* opt ci && update e2e badge by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/79
* update e2e badge url by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/80
* enable disable_ipv6 in advance by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/81
* add release-note & add badge & add github action by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/82
* typo by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/83
* opt validate overlaySubnet and overlaySubnet, and add unit tests by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/88
* fix vlanif ip mask by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/89
* add macvlan overlay e2e by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/86
* support ipv4-only and ipv6-only by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/91
* fix wrong cni-package url when 'RUN_ON_LOCAL'=false by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/93
* unit test by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/92
* create an issue when night ci failed by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/96
* unit-test1 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/97
* typo by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/98
* fix failed to get defaultOverlayIP by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/99
* create an issue by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/100
* add use spiderdoctor standalone e2e by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/101
* add spiderdoctor overlay e2e by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/102
* Update changelog and ready to release v0.1.9 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/103
* add design.md doc by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/104
* fix auto release by workflow_dispatch by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/105
* typo by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/106
* update changelog by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/112
* fix pod failed to visit host when host multi-nic by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/111
* y branch by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/115
* fix dockerfile wrong arch by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/116
* opt cmd del: skip call cmdDel when pod no ips by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/119
* update spiderdoctor version 0.3.0 by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/128
* unitest by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/123
* fix unit test by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/129
* using ghcr.io when run github e2e by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/131
* add unit-test by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/132
* found cali* or lxc* by using link.parentIndex without netlink.RouteGet by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/133
* v0.2.1 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/134
* fix ci: sleep for pod ready by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/138
* fix: overlay route miss net1 route by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/149
* enable ipv6 forwarding && revert last changes by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/154
* docs: add some readme.md by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/151
* support custom mac address by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/153
* add cilium e2e by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/141
* upload e2e spiderdoctor report by @ii2day in https://github.com/spidernet-io/cni-plugins/pull/155
* document supplement by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/157
* update doc by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/158
* add an option to only override mac address for NIC  created by Main CNI by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/160
* v0.2.2 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/161
* v0.2.2 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/162
* e2e ci: ensure crds installed first for multus-underlay by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/165
* Update Design.md by @wholj in https://github.com/spidernet-io/cni-plugins/pull/167
* fix doc typo by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/168
* bump golang version to 1.20 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/173
* support ip conflict checking by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/172
* bump chart to v0.2.3 by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/174
* fix action typo by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/175
* veth: opt cmdDel by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/179
* cmdDel: ignore pod'ns has gone by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/180
* add static neigh table for communicate between node and pod's underla… by @cyclinder in https://github.com/spidernet-io/cni-plugins/pull/182

## New Contributors
* @ii2day made their first contribution in https://github.com/spidernet-io/cni-plugins/pull/67
* @wholj made their first contribution in https://github.com/spidernet-io/cni-plugins/pull/167

**Full Changelog**: https://github.com/spidernet-io/cni-plugins/compare/v0.1.6...v0.2.5
