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
CILIUM_VERSION=${CILIUM_VERSION:-v0.12.0}

[ -z "${INSTALL_TIME_OUT}" ] && INSTALL_TIME_OUT=600s

export CALICO_VERSION=v3.24.0
export CALICO_AUTODETECTION_METHOD=interface=eth0
if [ ${RUN_ON_LOCAL} == "true" ]; then
  export CALICO_IMAGE_REPO=docker.m.daocloud.io
else
  export CALICO_IMAGE_REPO=docker.io
fi

mkdir -p ${PROJECT_ROOT_PATH}/.tmp/config
cp ${PROJECT_ROOT_PATH}/test/config/calico.yaml ${PROJECT_ROOT_PATH}/.tmp/config/calico.yaml

# Install default-cni
case ${DEFAULT_CNI} in
  calico|k8s-pod-network)
    echo "Install Calico with ${IP_FAMILY}"
    ;;
  cilium)
    echo "cilium as default cni is not support at present, exit 1"
    exit 1
    ;;
  *)
    echo "the value of 'DEFAULT_CNI' only be calico or cilium"
    exit 1
esac

case ${IP_FAMILY} in
  ipv4)
      export CALICO_CNI_ASSIGN_IPV4=true
      export CALICO_CNI_ASSIGN_IPV6=false
      export CALICO_IP_AUTODETECT=autodetect
      export CALICO_IP6_AUTODETECT=autodetect
      export CALICO_FELIX_IPV6SUPPORT=false
    ;;
  ipv6)
      export CALICO_CNI_ASSIGN_IPV4=false
      export CALICO_CNI_ASSIGN_IPV6=true
      export CALICO_IP_AUTODETECT=autodetect
      export CALICO_IP6_AUTODETECT=autodetect
      export CALICO_FELIX_IPV6SUPPORT=true
    ;;
  dual)
      export CALICO_CNI_ASSIGN_IPV4=true
      export CALICO_CNI_ASSIGN_IPV6=true
      export CALICO_IP_AUTODETECT=autodetect
      export CALICO_IP6_AUTODETECT=autodetect
      export CALICO_FELIX_IPV6SUPPORT=true
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
kubectl get po -n kube-system --kubeconfig ${E2E_KUBECONFIG}

echo -e "\033[35m Succeed to install Calico \033[0m"
