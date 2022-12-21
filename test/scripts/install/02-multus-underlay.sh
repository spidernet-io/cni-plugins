#！/bin/bash

set -o errexit -o nounset -o pipefail

CURRENT_FILENAME=$( basename $0 )
CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/../.. && pwd )

[  -z "${IP_FAMILY}" ] &&  echo "must be provide IP_FAMILY by using env IP_FAMILY" && exit 1
[  -z "${DEFAULT_CNI}" ] &&  echo "must be provide DEFAULT_CNI by using env DEFAULT_CNI" && exit 1
[ -z "${INSTALL_TIME_OUT}" ] && INSTALL_TIME_OUT=600s


#INSTALLED=` kubectl get po -n kube-system -l app.kubernetes.io/instance=multus-underlay --kubeconfig ${E2E_KUBECONFIG}`
#[ -n ${INSTALLED} ] && echo "Warning!! multus-underlay has been deployed, skip install multus-underlay" && exit 0

# Multus config
MULTUS_UNDERLAY_VERSION=${MULTUS_UNDERLAY_VERSION:-0.1.4}
MACVLAN_MASTER=${MACVLAN_MASTER:-eth0}
MACVLAN_TYPE=${MACVLAN_TYPE:-macvlan-overlay}

MULTUS_HELM_OPTIONS=" --set multus.config.cni_conf.clusterNetwork=${DEFAULT_CNI} \
--set macvlan.master=${MACVLAN_MASTER} \
--set macvlan.vlanID=0 \
--set macvlan.type=${MACVLAN_TYPE} \
--set macvlan.name=macvlan-overlay-vlan0 \
--set sriov.sriov_crd.vlanId=500 \
--set meta-plugins.image.tag=latest \
--set sriov.manifests.enable=true
"

case ${IP_FAMILY} in
  ipv4)
    MULTUS_HELM_OPTIONS+=" --set cluster_subnet.service_subnet.ipv4=${CLUSTER_SERVICE_SUBNET_V4} \
    --set cluster_subnet.pod_subnet="{${CLUSTER_POD_SUBNET_V4}}" "
    SERVICE_HIJACK_SUBNET="[\"${CLUSTER_SERVICE_SUBNET_V4}\"]"
    OVERLAY_HIJACK_SUBNET="[\"${CLUSTER_POD_SUBNET_V4}\"]"
    ;;
  ipv6)
    MULTUS_HELM_OPTIONS+=" --set cluster_subnet.service_subnet.ipv6=${CLUSTER_SERVICE_SUBNET_V6} \
    --set cluster_subnet.pod_subnet="{${CLUSTER_POD_SUBNET_V6}}" "
    SERVICE_HIJACK_SUBNET="[\"${CLUSTER_SERVICE_SUBNET_V6}\"]"
    OVERLAY_HIJACK_SUBNET="[\"${CLUSTER_POD_SUBNET_V6}\"]"
    ;;
  dual)
    MULTUS_HELM_OPTIONS+=" --set cluster_subnet.service_subnet.ipv4=${CLUSTER_SERVICE_SUBNET_V4}  \
    --set cluster_subnet.service_subnet.ipv6=${CLUSTER_SERVICE_SUBNET_V6} \
    --set "cluster_subnet.pod_subnet={${CLUSTER_POD_SUBNET_V4},${CLUSTER_POD_SUBNET_V6}}" "
    SERVICE_HIJACK_SUBNET="[\"${CLUSTER_SERVICE_SUBNET_V4}\",\"${CLUSTER_SERVICE_SUBNET_V6}\"]"
    OVERLAY_HIJACK_SUBNET="[\"${CLUSTER_POD_SUBNET_V4}\",\"${CLUSTER_POD_SUBNET_V6}\"]"
    ;;
  *)
    echo "the value of IP_FAMILY: ipv4 or ipv6 or dual"
    exit 1
esac

if [ ${RUN_ON_LOCAL} == false ]; then
  MULTUS_HELM_OPTIONS+=" --set multus.image.repository=ghcr.io/k8snetworkplumbingwg/multus-cni \
  --set sriov.images.sriovCni.repository=ghcr.io/k8snetworkplumbingwg/sriov-cni \
  --set sriov.images.sriovDevicePlugin.repository=ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin "
fi

if [ -n ${META_PLUGINS_CI_REPO} ] ;then
  MULTUS_HELM_OPTIONS+=" --set meta-plugins.image.repository=${META_PLUGINS_CI_REPO} "
fi

if [ -n ${META_PLUGINS_CI_TAG} ] ;then
  MULTUS_HELM_OPTIONS+=" --set meta-plugins.image.tag=${META_PLUGINS_CI_TAG} "
fi

echo "MULTUS_HELM_OPTIONS: ${MULTUS_HELM_OPTIONS}"

helm repo add daocloud https://daocloud.github.io/dce-charts-repackage/
# prepare image
HELM_IMAGES_LIST=` helm template test daocloud/multus-underlay --version ${MULTUS_UNDERLAY_VERSION} ${MULTUS_HELM_OPTIONS} | grep " image: " | tr -d '"'| awk '{print $2}' `

[ -z "${HELM_IMAGES_LIST}" ] && echo "can't found image of multus-underlay" && exit 1
LOCAL_IMAGE_LIST=`docker images | awk '{printf("%s:%s\n",$1,$2)}'`

for IMAGE in ${HELM_IMAGES_LIST}; do
  found=false
  for LOCAL_IMAGE in ${LOCAL_IMAGE_LIST}; do
    if [ "${IMAGE}" == "${LOCAL_IMAGE}" ]; then
        found=true
    fi
  done
  if [ "${found}" == "false" ] ; then
      echo "===> docker pull ${IMAGE}..."
      docker pull ${IMAGE}
  fi
  echo "===> load image ${IMAGE} to kind..."
  kind load docker-image ${IMAGE} --name ${IP_FAMILY}
done

# helm repo update daocloud
helm install multus-underlay daocloud/multus-underlay -n kube-system  --kubeconfig ${E2E_KUBECONFIG} ${MULTUS_HELM_OPTIONS} --version ${MULTUS_UNDERLAY_VERSION}

# wait multus-ready
kubectl wait --for=condition=ready -l app.kubernetes.io/instance=multus-underlay --timeout=${INSTALL_TIME_OUT} pod -n kube-system --kubeconfig ${E2E_KUBECONFIG}

# create extra multus cr to test
cat <<EOF | kubectl --kubeconfig ${E2E_KUBECONFIG} apply -f -
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  annotations:
    v1.multus-underlay-cni.io/coexist-types: '["macvlan-standalone"]'
    v1.multus-underlay-cni.io/default-cni: "true"
    v1.multus-underlay-cni.io/instance-type: macvlan_standalone
    v1.multus-underlay-cni.io/underlay-cni: "true"
    v1.multus-underlay-cni.io/vlanId: "${MACVLAN_VLANID}"
  labels:
    v1.multus-underlay-cni.io/instance-status: enable
  name: macvlan-standalone-vlan${MACVLAN_VLANID}
  namespace: kube-system
spec:
  config: |-
    {
        "cniVersion": "0.3.1",
        "name": "macvlan-overlay",
        "plugins": [
            {
                "type": "macvlan",
                "master": "eth0.${MACVLAN_VLANID}",
                "mode": "bridge",
                "ipam": {
                    "type": "spiderpool",
                    "log_level": "DEBUG"
                }
            },{
                "type": "veth",
                "service_hijack_subnet": ${SERVICE_HIJACK_SUBNET},
                "overlay_hijack_subnet": ${OVERLAY_HIJACK_SUBNET},
                "additional_hijack_subnet": [],
                "migrate_route": -1,
                "rp_filter": {
                    "set_host": true,
                    "value": 2
                },
                "overlay_interface": "eth0",
                "skip_call": false
            }
        ]
    }
---
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  annotations:
    v1.multus-underlay-cni.io/coexist-types: '["macvlan-overlay"]'
    v1.multus-underlay-cni.io/default-cni: "false"
    v1.multus-underlay-cni.io/instance-type: macvlan_overlay
    v1.multus-underlay-cni.io/underlay-cni: "true"
    v1.multus-underlay-cni.io/vlanId: "${MACVLAN_VLANID}"
  labels:
    v1.multus-underlay-cni.io/instance-status: enable
  name: macvlan-overlay-vlan${MACVLAN_VLANID}
  namespace: kube-system
spec:
  config: |-
    {
        "cniVersion": "0.3.1",
        "name": "macvlan-overlay",
        "plugins": [
            {
                "type": "macvlan",
                "master": "eth0.${MACVLAN_VLANID}",
                "mode": "bridge",
                "ipam": {
                    "type": "spiderpool",
                    "log_level": "DEBUG"
                }
            },{
                "type": "router",
                "service_hijack_subnet": ${SERVICE_HIJACK_SUBNET},
                "overlay_hijack_subnet": ${OVERLAY_HIJACK_SUBNET},
                "additional_hijack_subnet": [],
                "migrate_route": -1,
                "rp_filter": {
                    "set_host": true,
                    "value": 2
                },
                "overlay_interface": "eth0",
                "skip_call": false
            }
        ]
    }
EOF

echo -e "\033[35m Succeed to install Multus-underlay \033[0m"
