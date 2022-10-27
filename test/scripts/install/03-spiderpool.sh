#!/bin/bash

set -o errexit -o nounset -o pipefail

#INSTALLED=` helm list -n kube-system --kubeconfig ${E2E_KUBECONFIG} | awk '{print $1}'| grep spiderpool `
#[ -n ${INSTALLED} ] && echo "Warning!! spiderpool has been deployed, skip install spiderpool" && exit 0

SPIDERPOOL_VERSION=${SPIDERPOOL_VERSION:-0.2.0}
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

[ -z ${INSTALL_TIME_OUT} ] && INSTALL_TIME_OUT=600s

SPIDERPOOL_HELM_OPTIONS=""
case ${IP_FAMILY} in
  ipv4)
    SPIDERPOOL_HELM_OPTIONS+=" --set spiderpool.feature.enableIPv4=true \
    --set spiderpool.feature.enableSpiderSubnet=false \
    --set spiderpool.clusterDefaultPool.installIPv4IPPool=true \
    --set spiderpool.clusterDefaultPool.ipv4Subnet=${SPIDERPOOL_DEFAULT_POOL_V4} \
    --set spiderpool.clusterDefaultPool.ipv4IPRanges={${SPIDERPOOL_IP_RANGES_V4}} \
    --set spiderpool.clusterDefaultPool.ipv4Gateway=${SPIDERPOOL_DEFAULT_GATEWAY_V4}"
    ;;
  ipv6)
    SPIDERPOOL_HELM_OPTIONS+=" --set spiderpool.feature.enableIPv4=false \
    --set spiderpool.feature.enableSpiderSubnet=false \
    --set spiderpool.feature.enableIPv6=true \
    --set spiderpool.clusterDefaultPool.installIPv4IPPool=false \
    --set spiderpool.clusterDefaultPool.installIPv6IPPool=true \
    --set spiderpool.clusterDefaultPool.ipv6Subnet=${SPIDERPOOL_DEFAULT_POOL_V6} \
    --set spiderpool.clusterDefaultPool.ipv6IPRanges={${SPIDERPOOL_IP_RANGES_V6}} \
    --set spiderpool.clusterDefaultPool.ipv6Gateway=${SPIDERPOOL_DEFAULT_GATEWAY_V6}"
    ;;
  dual)
    SPIDERPOOL_HELM_OPTIONS+=" --set spiderpool.feature.enableIPv4=true \
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

echo "SPIDERPOOL_HELM_OPTIONS: ${SPIDERPOOL_HELM_OPTIONS}"

helm repo add daocloud https://daocloud.github.io/network-charts-repackage/

HELM_IMAGES_LIST=` helm template test daocloud/spiderpool --version ${SPIDERPOOL_VERSION} ${SPIDERPOOL_HELM_OPTIONS} | grep " image: " | tr -d '"'| awk '{print $2}' `

[ -z "${HELM_IMAGES_LIST}" ] && echo "can't found image of spiderpool" && exit 1
LOCAL_IMAGE_LIST=`docker images | awk '{printf("%s:%s\n",$1,$2)}'`

for IMAGE in ${HELM_IMAGES_LIST}; do
  found=false
  for LOCAL_IMAGE in ${LOCAL_IMAGE_LIST}; do
    if [ "${IMAGE}" == "${LOCAL_IMAGE}" ]; then
        found=true
    fi
  done
  if [ "${found}" == "false" ] ; then
      echo "===> docker pull ${IMAGE}... "
      docker pull ${IMAGE}
  fi
  echo "===> load image ${IMAGE} to kind..."
  kind load docker-image ${IMAGE} --name ${IP_FAMILY}
done

# Install spiderpool
helm install spiderpool daocloud/spiderpool --wait -n kube-system --kubeconfig ${E2E_KUBECONFIG} ${SPIDERPOOL_HELM_OPTIONS} --version ${SPIDERPOOL_VERSION}
kubectl wait --for=condition=ready -l app.kubernetes.io/name=spiderpool --timeout=${INSTALL_TIME_OUT} pod -n kube-system \
--kubeconfig ${E2E_KUBECONFIG}

cat <<EOF | kubectl --kubeconfig ${E2E_KUBECONFIG} apply -f -
apiVersion: spiderpool.spidernet.io/v1
kind: SpiderIPPool
metadata:
  name: vlan${MACVLAN_VLANID}-v4
spec:
  disable: false
  gateway: ${SPIDERPOOL_VLAN100_GATEWAY_V4}
  ipVersion: 4
  ips:
  - ${SPIDERPOOL_VLAN100_RANGES_V4}
  subnet: ${SPIDERPOOL_VLAN100_POOL_V4}
  vlan: ${MACVLAN_VLANID}
---
apiVersion: spiderpool.spidernet.io/v1
kind: SpiderIPPool
metadata:
  name: vlan100-v6
spec:
  disable: false
  gateway: ${SPIDERPOOL_VLAN100_GATEWAY_V6}
  ipVersion: 6
  ips:
  - ${SPIDERPOOL_VLAN100_RANGES_V6}
  subnet: ${SPIDERPOOL_VLAN100_POOL_V6}
  vlan: ${MACVLAN_VLANID}
EOF

echo -e "\033[35m Succeed to install spiderpool \033[0m"