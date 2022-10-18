#!/bin/bash

set -o errexit -o nounset -o pipefail

SPIDERPOOL_VERSION=${SPIDERPOOL_VERSION:-v0.2.0}
SPIDERPOOL_DEFAULT_POOL_V4=172.18.0.0/16
SPIDERPOOL_DEFAULT_POOL_V6=fc00:f853:ccd:e793::/64
SPIDERPOOL_IP_RANGES_V4=172.18.1.100-172.18.100.254
SPIDERPOOL_IP_RANGES_V6=fc00:f853:ccd:e793::100-fc00:f853:ccd:e793::fff1
SPIDERPOOL_DEFAULT_GATEWAY_V4=172.18.0.1
SPIDERPOOL_DEFAULT_GATEWAY_V6=fc00:f853:ccd:e793::1
SPIDERPOOL_VLAN100_POOL_V4=172.100.0.0/16
SPIDERPOOL_VLAN100_POOL_V6=fd00:172:100::/64
SPIDERPOOL_VLAN100_RANGES_V4=172.100.0.201-172.100.10.199
SPIDERPOOL_VLAN100_RANGES_V6=fd00:172:100::201-fd00:172:100::fff1
SPIDERPOOL_VLAN100_GATEWAY_V4=172.100.0.1
SPIDERPOOL_VLAN100_GATEWAY_V6=fd00:172:100::1

[ -z ${INSTALL_TIME_OUT} ] ; then INSTALL_TIME_OUT=600s ; fi

SPIDERPOL_HELM_OPTIONS=""
case ${IP_FAMILY} in
  ipv4)
    SPIDERPOL_HELM_OPTIONS+=" --set spiderpool.feature.enableIPv4=true \
    --set spiderpool.feature.enableSpiderSubnet=false \
    --set spiderpool.clusterDefaultPool.installIPv4IPPool=true \
    --set spiderpool.clusterDefaultPool.ipv4Subnet=${SPIDERPOOL_DEFAULT_POOL_V4} \
    --set spiderpool.clusterDefaultPool.ipv4IPRanges={${SPIDERPOOL_IP_RANGES_V4}} \
    --set spiderpool.clusterDefaultPool.ipv4Gateway=${SPIDERPOOL_DEFAULT_GATEWAY_V4}"
    ;;
  ipv6)
    SPIDERPOL_HELM_OPTIONS+=" --set spiderpool.feature.enableIPv4=false \
    --set spiderpool.feature.enableSpiderSubnet=false \
    --set spiderpool.feature.enableIPv6=true \
    --set spiderpool.clusterDefaultPool.installIPv4IPPool=false \
    --set spiderpool.clusterDefaultPool.installIPv6IPPool=true \
    --set spiderpool.clusterDefaultPool.ipv6Subnet=${SPIDERPOOL_DEFAULT_POOL_V6} \
    --set spiderpool.clusterDefaultPool.ipv6IPRanges={${SPIDERPOOL_IP_RANGES_V6}} \
    --set spiderpool.clusterDefaultPool.ipv6Gateway=${SPIDERPOOL_DEFAULT_GATEWAY_V6}"
    ;;
  dual)
    SPIDERPOL_HELM_OPTIONS+=" --set spiderpool.feature.enableIPv4=true \
    --set spiderpool.feature.enableIPv6=true \
    --set spiderpool.feature.enableSpiderSubnet=false \
    --set spiderpool.clusterDefaultPool.installIPv4IPPool=true \
    --set spiderpool.clusterDefaultPool.installIPv6IPPool=true \
    --set spiderpool.clusterDefaultPool.ipv4Subnet=${SPIDERPOOL_DEFAULT_POOL_V4} \
    --set spiderpool.clusterDefaultPool.ipv6Subnet=${SPIDERPOOL_DEFAULT_POOL_V6} \
    --set spiderpool.clusterDefaultPool.ipv4IPRanges={${SPIDERPOOL_IP_RANGES_V4}} \
    --set spiderpool.clusterDefaultPool.ipv6IPRanges={${SPIDERPOOL_IP_RANGES_V6}} \
    --set spiderpool.clusterDefaultPool.ipv4Gateway=${SPIDERPOOL_DEFAULT_GATEWAY_V4}  \
    --set spiderpool.clusterDefaultPool.ipv6Gateway=${SPIDERPOOL_DEFAULT_GATEWAY_V6} "
    ;;
  *)
    echo "the value of IP_FAMILY: ipv4 or ipv6 or dual"
    exit 1
esac

echo "SPIDERPOOL_HELM_OPTIONS: ${SPIDERPOL_HELM_OPTIONS}"

# Install spiderpool
helm install spiderpool daocloud/spiderpool -n kube-system --kubeconfig ${E2E_KUBECONFIG} ${SPIDERPOL_HELM_OPTIONS} --version ${SPIDERPOOL_VERSION}
kubectl wait --for=condition=ready -l app.kubernetes.io/name=spiderpool --timeout=${INSTALL_TIME_OUT} pod -n kube-system \
--kubeconfig ${E2E_KUBECONFIG}

cat <<EOF | kubectl --kubeconfig ${E2E_KUBECONFIG} apply -f -
apiVersion: spiderpool.spidernet.io/v1
kind: SpiderIPPool
metadata:
  name: vlan${MULTUS_SECOND_VLAN}-v4
spec:
  disable: false
  gateway: ${SPIDERPOOL_VLAN100_GATEWAY_V4}
  ipVersion: 4
  ips:
  - ${SPIDERPOOL_VLAN100_RANGES_V4}
  subnet: ${SPIDERPOOL_VLAN100_POOL_V4}
  vlan: 100
---
apiVersion: spiderpool.spidernet.io/v1
kind: SpiderIPPool
metadata:
  name: VLAN100-v6
spec:
  disable: false
  gateway: ${SPIDERPOOL_VLAN100_GATEWAY_V6}
  ipVersion: 6
  ips:
  - ${SPIDERPOOL_VLAN100_RANGES_V6}
  subnet: ${SPIDERPOOL_VLAN100_POOL_V6}
  vlan: 100
EOF
