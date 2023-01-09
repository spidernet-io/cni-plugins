#!/bin/bash

set -o errexit -o nounset -o pipefail

OS=$(uname | tr 'A-Z' 'a-z')
SED_COMMAND=sed

CURRENT_FILENAME=$( basename $0 )
CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/../../.. && pwd )

#CALICO_POD_NUM=` kubectl get po -n kube-system -l k8s-app=calico-node --kubeconfig ${E2E_KUBECONFIG} `
#[ -n "${CALICO_POD_NUM}" ] && echo "Warning!! calico has been deployed, skip install calico" && exit 0

# Multus config
DEFAULT_CNI=${DEFAULT_CNI:-calico}
CILIUM_VERSION=${CILIUM_VERSION:-v1.12.5}

[ -z "${INSTALL_TIME_OUT}" ] && INSTALL_TIME_OUT=600s

export CALICO_VERSION=v3.24.0
export CALICO_AUTODETECTION_METHOD=interface=eth0

function install_calico() {
    if [ ${RUN_ON_LOCAL} == "true" ]; then
      export CALICO_IMAGE_REPO=docker.m.daocloud.io
    else
      export CALICO_IMAGE_REPO=docker.io
    fi

    mkdir -p ${PROJECT_ROOT_PATH}/.tmp/config
    cp ${PROJECT_ROOT_PATH}/test/config/calico.yaml ${PROJECT_ROOT_PATH}/.tmp/config/calico.yaml

    case ${IP_FAMILY} in
      ipv4)
          export CALICO_CNI_ASSIGN_IPV4=true
          export CALICO_CNI_ASSIGN_IPV6=false
          export CALICO_IP_AUTODETECT=autodetect
          export CALICO_IP6_AUTODETECT=autodetect
          export CALICO_FELIX_IPV6SUPPORT=false
          export CALICO_IPV6POOL_VXLAN=Never
        ;;
      ipv6)
          export CALICO_CNI_ASSIGN_IPV4=false
          export CALICO_CNI_ASSIGN_IPV6=true
          export CALICO_IP_AUTODETECT=autodetect
          export CALICO_IP6_AUTODETECT=autodetect
          export CALICO_FELIX_IPV6SUPPORT=true
          export CALICO_IPV6POOL_VXLAN=Always
        ;;
      dual)
          export CALICO_CNI_ASSIGN_IPV4=true
          export CALICO_CNI_ASSIGN_IPV6=true
          export CALICO_IP_AUTODETECT=autodetect
          export CALICO_IP6_AUTODETECT=autodetect
          export CALICO_FELIX_IPV6SUPPORT=true
          export CALICO_IPV6POOL_VXLAN=Always
        ;;
      *)
        echo "the value of IP_FAMILY: ipv4 or ipv6 or dual"
        exit 1
    esac

    if [ ${OS} == "darwin" ]; then SED_COMMAND=gsed ; fi

    ENV_LIST=`env | egrep "^CALICO_" `
    for env in ${ENV_LIST}; do
        KEY="${env%%=*}"
        VALUE="${env#*=}"
        echo $KEY $VALUE
        ${SED_COMMAND} -i "s/<<${KEY}>>/${VALUE}/g" ${PROJECT_ROOT_PATH}/.tmp/config/calico.yaml
    done


    CALICO_IMAGE_LIST=`cat ${PROJECT_ROOT_PATH}/.tmp/config/calico.yaml | grep 'image: ' | tr -d '"' | awk '{print $2}'`
    [ -z "${CALICO_IMAGE_LIST}" ] && echo "can't found image of calico" && exit 1
    LOCAL_IMAGE_LIST=`docker images | awk '{printf("%s:%s\n",$1,$2)}'`

    for CALICO_IMAGE in ${CALICO_IMAGE_LIST}; do
      found=false
      for LOCAL_IMAGE in ${LOCAL_IMAGE_LIST}; do
        if [ "${CALICO_IMAGE}" == "${LOCAL_IMAGE}" ]; then
            found=true
        fi
      done
      if [ "${found}" == "false" ] ; then
          echo "===> docker pull ${CALICO_IMAGE} "
          docker pull ${CALICO_IMAGE}
      fi
      echo "===> load image ${CALICO_IMAGE} to kind..."
      kind load docker-image ${CALICO_IMAGE} --name ${IP_FAMILY}
    done

    kubectl apply -f  ${PROJECT_ROOT_PATH}/.tmp/config/calico.yaml --kubeconfig ${E2E_KUBECONFIG}

    sleep 3

    kubectl wait --for=condition=ready -l k8s-app=calico-node --timeout=${INSTALL_TIME_OUT} pod -n kube-system --kubeconfig ${E2E_KUBECONFIG}

    echo -e "\033[35m Succeed to install calico \033[0m"
}

function install_cilium() {
    # cni.exclusive using multus-cni need close
    # kubeProxyReplacement Enhance kube-proxy (value probe static defult: probe)
    # k8sServiceHost api-server address
    # k8sServicePort api-service port
    # bpf.vlanBypass allow vlan traffic to pass
    CILIUM_HELM_OPTIONS=" --set cni.exclusive=false \
                          --set kubeProxyReplacement=probe \
                          --set k8sServiceHost=${IP_FAMILY}-control-plane \
                          --set k8sServicePort=6443 \
                          --set bpf.vlanBypass=0 "
    case ${IP_FAMILY} in
      ipv4)
          CILIUM_HELM_OPTIONS+=" --set ipam.operator.clusterPoolIPv4PodCIDR=${CLUSTER_POD_SUBNET_V4} \
                                 --set ipv4.enabled=true \
                                 --set ipv6.enabled=false "
        ;;
      ipv6)
          CILIUM_HELM_OPTIONS+=" --set ipam.operator.clusterPoolIPv6PodCIDR=${CILIUM_CLUSTER_POD_SUBNET_V6} \
                                 --set ipv4.enabled=false \
                                 --set ipv6.enabled=true \
                                 --set tunnel=disabled \
                                 --set ipv6NativeRoutingCIDR=${CILIUM_CLUSTER_POD_SUBNET_V6} \
                                 --set autoDirectNodeRoutes=true \
                                 --set enableIPv6Masquerade=true "
        ;;
      dual)
          CILIUM_HELM_OPTIONS+=" --set ipam.operator.clusterPoolIPv4PodCIDR=${CLUSTER_POD_SUBNET_V4} \
                                 --set ipam.operator.clusterPoolIPv6PodCIDR=${CILIUM_CLUSTER_POD_SUBNET_V6} \
                                 --set ipv4.enabled=true \
                                 --set ipv6.enabled=true "
        ;;
      *)
        echo "the value of IP_FAMILY: ipv4 or ipv6 or dual"
        exit 1
    esac

  echo "CILIUM_HELM_OPTIONS: ${CILIUM_HELM_OPTIONS}"
#  if [ ${RUN_ON_LOCAL} == true ]; then
#    CILIUM_HELM_OPTIONS+=" --set image.repository=ghcr.m.daocloud.io/cilium/cilium "
#  fi
  helm repo add cilium https://helm.cilium.io

#  HELM_IMAGES_LIST=` helm template test cilium/cilium --version ${CILIUM_VERSION} ${CILIUM_HELM_OPTIONS} | grep " image: " | tr -d '"'| awk '{print $2}' | uniq `
#
#  [ -z "${HELM_IMAGES_LIST}" ] && echo "can't found image of spiderpool" && exit 1
#  LOCAL_IMAGE_LIST=`docker images | awk '{printf("%s:%s\n",$1,$2)}'`
#
#  for IMAGE in ${HELM_IMAGES_LIST}; do
#    found=false
#    for LOCAL_IMAGE in ${LOCAL_IMAGE_LIST}; do
#      if [ "${IMAGE}" == "${LOCAL_IMAGE}" ]; then
#          found=true
#      fi
#    done
#    if [ "${found}" == "false" ] ; then
#        echo "===> docker pull ${IMAGE}... "
#        docker pull ${IMAGE}
#    fi
#    echo "===> load image ${IMAGE} to kind..."
#    kind load docker-image ${IMAGE} --name ${IP_FAMILY}
#  done

  # Install cilium
  helm install cilium cilium/cilium --wait -n kube-system --kubeconfig ${E2E_KUBECONFIG} ${CILIUM_HELM_OPTIONS} --version ${CILIUM_VERSION}
  kubectl wait --for=condition=ready -l k8s-app=cilium --timeout=${INSTALL_TIME_OUT} pod -n kube-system \
  --kubeconfig ${E2E_KUBECONFIG}

  sleep 10

  echo -e "\033[35m Succeed to install cilium \033[0m"
}

    # Install default-cni
case ${DEFAULT_CNI} in
  calico|k8s-pod-network)
    echo "Install calico with ${IP_FAMILY}"
    install_calico
    ;;
  cilium)
    echo "Install cilium with ${IP_FAMILY}"
    install_cilium
    ;;
  *)
    echo "the value of 'DEFAULT_CNI' only be calico or cilium"
    exit 1
esac

kubectl get po -n kube-system --kubeconfig ${E2E_KUBECONFIG} -owide