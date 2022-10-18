#!/bin/bash

set -o errexit -o nounset -o pipefail

exit 0
OS=$(uname | tr 'A-Z' 'a-z')
SED_COMMAND=sed

CURRENT_FILENAME=$( basename $0 )
CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/../../.. && pwd )

# Multus config
DEFAULT_CNI=${DEFAULT_CNI:-calico}
CILIUM_VERSION=${CILIUM_VERSION:-v0.12.0}

[ -z ${INSTALL_TIME_OUT} ] ; then INSTALL_TIME_OUT=600s ; fi

export CALICO_VERSION=v3.24.0
if [ ${RUN_ON_LOCAL} == "true" ]; then
  export CALICO_IMAGE_REPO=docker.m.daocloud.io
else
  export CALICO_IMAGE_REPO=docker.io
fi

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
      export CALICO_IP6_AUTODETECT=none
      export FELIX_IPV6SUPPORT=false
    ;;
  ipv6)
      export CALICO_CNI_ASSIGN_IPV4=false
      export CALICO_CNI_ASSIGN_IPV6=true
      export CALICO_IP_AUTODETECT=none
      export CALICO_IP6_AUTODETECT=autodetect
      export FELIX_IPV6SUPPORT=true
    ;;
  dual)
      export CALICO_CNI_ASSIGN_IPV4=true
      export CALICO_CNI_ASSIGN_IPV6=true
      export CALICO_IP_AUTODETECT=autodetect
      export CALICO_IP6_AUTODETECT=autodetect
      export FELIX_IPV6SUPPORT=true
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

kubectl apply -f  ${PROJECT_ROOT_PATH}/.tmp/config/calico.yaml --kubeconfig ${E2E_KUBECONFIG}
kubectl wait --for=condition=ready -l k8s-app=calico-node --timeout=${INSTALL_TIME_OUT} pod -n kube-system --kubeconfig ${E2E_KUBECONFIG}
#
echo -e "\033[35m Succeed to install Calico \033[0m"
