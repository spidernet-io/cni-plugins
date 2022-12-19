#!/bin/bash

set -o errexit -o nounset -o pipefail

SPIDERDOCTOR_VERSION=${SPIDERDOCTOR_VERSION:-0.2.1}

[ -z ${INSTALL_TIME_OUT} ] && INSTALL_TIME_OUT=600s

SPIDERDOCTOR_HELM_OPTIONS=" "
case ${IP_FAMILY} in
  ipv4)
    SPIDERDOCTOR_HELM_OPTIONS+=" --set feature.enableIPv4=true \
    --set feature.enableIPv6=false \
    --set feature.aggregateReport.enabled=false"
    ;;
  ipv6)
    SPIDERDOCTOR_HELM_OPTIONS+=" --set feature.enableIPv4=false \
    --set feature.enableIPv6=true \
    --set feature.aggregateReport.enabled=false"
    ;;
  dual)
    SPIDERDOCTOR_HELM_OPTIONS+=" --set feature.enableIPv4=true \
    --set feature.enableIPv6=true \
    --set feature.aggregateReport.enabled=false"
    ;;
  *)
    echo "the value of IP_FAMILY: ipv4 or ipv6 or dual"
    exit 1
esac

if [ ${RUN_ON_LOCAL} == true ]; then
  SPIDERDOCTOR_HELM_OPTIONS+=" --set spiderdoctorAgent.image.registry=ghcr.m.daocloud.io \
   --set spiderdoctorController.image.registry=ghcr.m.daocloud.io "
fi

echo "SPIDERDOCTOR_HELM_OPTIONS: ${SPIDERDOCTOR_HELM_OPTIONS}"

helm repo add spiderdoctor https://spidernet-io.github.io/spiderdoctor
helm repo update
HELM_IMAGES_LIST=` helm template test spiderdoctor/spiderdoctor --version ${SPIDERDOCTOR_VERSION} ${SPIDERDOCTOR_HELM_OPTIONS} | grep " image: " | tr -d '"'| awk '{print $2}' `

[ -z "${HELM_IMAGES_LIST}" ] && echo "can't found image of SPIDERDOCTOR" && exit 1
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

# Install SPIDERDOCTOR
helm install spiderdoctor spiderdoctor/spiderdoctor -n kube-system --kubeconfig ${E2E_KUBECONFIG} ${SPIDERDOCTOR_HELM_OPTIONS} --version ${SPIDERDOCTOR_VERSION}
kubectl wait --for=condition=ready -l app.kubernetes.io/name=spiderdoctor --timeout=${INSTALL_TIME_OUT} pod -n kube-system \
--kubeconfig ${E2E_KUBECONFIG}

echo -e "\033[35m Succeed to install spiderdoctor \033[0m"