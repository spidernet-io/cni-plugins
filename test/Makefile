include Makefile.defs

.PHONY: test
test: kind-init install vlan update-repo e2e

.PHONY: vlan
vlan:
	export VLAN_GATEWAY_CONTAINER=${VLAN_GATEWAY_CONTAINER} ; \
	export IP_FAMILY=${IP_FAMILY} ; \
	bash scripts/network-config.sh

.PHONY: e2e
e2e:
	export E2E_KUBECONFIG_PATH=${E2E_KUBECONFIG}; [ -f "$${E2E_KUBECONFIG_PATH}" ] || { echo "error, does not exist KUBECONFIG: ${E2E_KUBECONFIG_PATH}" ; exit 1 ; } ; \
	export E2E_CLUSTER_NAME=${IP_FAMILY} ; \
	export VLAN_GATEWAY_CONTAINER=${VLAN_GATEWAY_CONTAINER} ; \
	NODE_LIST="` docker ps | egrep " kindest/node.* $(IP_FAMILY)-(control|worker)" | awk '{print $$1 }' ` "; \
	[ -n "$$NODE_LIST" ] || { echo "error, failed to find any kind nodes, please setup kind cluster $(E2E_CLUSTER_NAME) first" ; exit 1 ; } ; \
	NODE_LIST=` echo "$${NODE_LIST}" | tr -d ' ' | tr '\n' ',' ` ; \
	NODE_LIST=$${NODE_LIST%%,} ; \
	echo "find cluster node: $${NODE_LIST}" ; \
	export E2E_KIND_CLUSTER_NODE_LIST="$${NODE_LIST}" ; \
	if [ "${IP_FAMILY}" = "ipv4" ] ; then \
	  export E2E_IPV4_ENABLED=true ; export E2E_IPV6_ENABLED=false ; \
	elif [ "${IP_FAMILY}" = "ipv6" ] ; then \
	  export E2E_IPV4_ENABLED=false ; export E2E_IPV6_ENABLED=true ; \
	else \
	  export E2E_IPV4_ENABLED=true ; export E2E_IPV6_ENABLED=true ; \
	fi ; \
	ginkgo --race --timeout=${E2E_TIMEOUT} --output-interceptor-mode=none --json-report e2ereport.json --output-dir ${ROOT_DIR}/../.tmp \
	 --label-filter="${E2E_GINKGO_LABELS}"  -randomize-suites -randomize-all  -vv --fail-fast ${E2E_GINKGO_OPTIONS} -r e2e/*

.PHONY: install
install:
	echo -e "  \033[35m Start install network-component: \033[0m" ; \
	if [ ! -f ${E2E_KUBECONFIG} ]; then echo "can't found ${E2E_KUBECONFIG}, please provide the right kubeconfig path using env E2E_KUBECONFIG" && exit 1; fi ; \
	export CLUSTER_POD_SUBNET_V4=${CLUSTER_POD_SUBNET_V4}; \
	export CLUSTER_POD_SUBNET_V6=${CLUSTER_POD_SUBNET_V6}; \
	export CLUSTER_SERVICE_SUBNET_V4=${CLUSTER_SERVICE_SUBNET_V4}; \
	export CLUSTER_SERVICE_SUBNET_V6=${CLUSTER_SERVICE_SUBNET_V6}; \
	export CALICO_VERSION=${CALICO_VERSION}; \
	export E2E_KUBECONFIG=${E2E_KUBECONFIG}; \
	export IP_FAMILY=${IP_FAMILY} ; \
	export DEFAULT_CNI=${DEFAULT_CNI} ; \
	export MACVLAN_VLANID=100 ; \
	export INSTALL_TIME_OUT=${INSTALL_TIME_OUT} ; \
	export VLAN_GATEWAY_CONTAINER=${VLAN_GATEWAY_CONTAINER} ; \
	export CNI_PLUGINS_VERSION=${CNI_PLUGINS_VERSION} ; \
	export RUN_ON_LOCAL=${RUN_ON_LOCAL} ; \
	export META_PLUGINS_CI_REPO=${META_PLUGINS_CI_REPO} ; \
	export META_PLUGINS_CI_TAG=${META_PLUGINS_CI_TAG} ; \
	export SPIDERPOOL_VERSION=${SPIDERPOOL_VERSION} ; \
	export SPIDERDOCTOR_VERSION=${SPIDERDOCTOR_VERSION} ; \
	for script in `ls scripts/install/` ; do \
	  echo "Running $${script}"  ; \
	  chmod +x scripts/install/$${script} ; \
	  bash scripts/install/$${script} || { echo "failed to run scripts/install/$${script} "; exit 1 ; } ; \
	done

.PHONY: kind-init
kind-init:
	export IP_FAMILY=${IP_FAMILY} ; \
	export KIND_NODE_TAG=${K8S_VERSION} ; \
	bash scripts/kind.sh

.PHONY: update-repo
update-repo:
	@NODE_LIST=` docker ps | egrep " kindest/node.* $(IP_FAMILY)-(control|worker)" | awk '{print $$1 }' ` ; \
	[ -n "$$NODE_LIST" ] || { echo "error, failed to find any kind nodes, please setup kind cluster $(IP_FAMILY) first" ; exit 1 ; } ; \
	for NODE in $${NODE_LIST} ; do \
	  docker cp scripts/update_repo.sh $${NODE}:/home/ ; \
	  docker exec $${NODE} chmod +x /home/update_repo.sh ; \
	  docker exec $${NODE} ./home/update_repo.sh ; \
	done

.PHONY: clean
clean:
	kind delete clusters ${IP_FAMILY}
	docker rm -f ${VLAN_GATEWAY_CONTAINER}
	rm -rf ../.tmp/
