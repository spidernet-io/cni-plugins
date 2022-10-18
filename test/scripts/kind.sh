#!/bin/sh

set -o errexit -o nounset -o xtrace

CURRENT_FILENAME=$( basename $0 )
CURRENT_DIR_PATH=$(cd $(dirname $0); pwd)
PROJECT_ROOT_PATH=$( cd ${CURRENT_DIR_PATH}/../.. && pwd )

# up a cluster with kind
create_cluster() {

  # Default Log level for all components in test clusters
  KIND_CLUSTER_LOG_LEVEL=${KIND_CLUSTER_LOG_LEVEL:-4}


  # create the network-config file
  cat <<EOF > "${PROJECT_ROOT_PATH}/.tmp/config/kind-config.yaml"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${IP_FAMILY:-dual}
networking:
  ipFamily: ${IP_FAMILY:-dual}
  kubeProxyMode: ${KUBE_PROXY_MODE:-iptables}
  disableDefaultCNI: true
nodes:
- role: control-plane
- role: worker
EOF

  kind create cluster \
    --image=kindest/node:${KIND_NODE_TAG:-v1.25.0}  \
    --wait=1m \
    -v=3 \
    --config=${PROJECT_ROOT_PATH}/.tmp/config/kind-config.yaml

  kubectl taint nodes ${IP_FAMILY}-control-plane  node-role.kubernetes.io/control-plane-
  echo "show kubernetes node " && docker ps
  echo "========================================================"
  echo "  kubectl get nodes -o wide  --kubeconfig ${KUBECONFIG}  "
  echo "========================================================"
}

main() {
  # ensure artifacts (results) directory exists when not in CI
  export ARTIFACTS=${PROJECT_ROOT_PATH}/.tmp/config
  mkdir -p ${ARTIFACTS}

  # export the KUBECONFIG to a unique path for testing
  KUBECONFIG=${ARTIFACTS}/.kube/config

  export KUBECONFIG=${KUBECONFIG}

  # create the cluster and run tests
  res=0
  create_cluster || res=$?

  exit $res
}

main