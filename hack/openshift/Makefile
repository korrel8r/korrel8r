# Deploy operators and resources for observability components.

OPERATOR_NAMESPACES=$(shell grep -h namespace: config/operators/*.yaml | sort -u | cut -d: -f2)

all: operators resources

operators:
	oc apply -k config/operators
	../wait-for-csv.sh openshift-logging cluster-logging
	../wait-for-csv.sh openshift-operators-redhat loki-operator
	../wait-for-csv.sh openshift-netobserv-operator netobserv-operator
	oc patch consoles.operator.openshift.io/cluster --type=merge --patch '{ "spec": { "plugins": ["monitoring-plugin", "logging-view-plugin", "netobserv-plugin"]}}'

STORAGE_CLASS?=$(shell oc get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}')

MINIO_ROLLOUT=../wait-rollout.sh -n minio deployment/minio

LOGGING_ROLLOUT=../wait-rollout.sh -n openshift-logging	\
deployment.apps/cluster-logging-operator		\
deployment.apps/logging-loki-distributor		\
deployment.apps/logging-loki-gateway			\
deployment.apps/logging-loki-querier			\
deployment.apps/logging-loki-query-frontend		\
deployment.apps/logging-view-plugin


NETOBSERV_ROLLOUT=../wait-rollout.sh -n netobserv	\
deployment.apps/loki-distributor			\
deployment.apps/loki-gateway				\
deployment.apps/loki-querier				\
deployment.apps/loki-query-frontend			\
deployment.apps/netobserv-plugin

resources:
	echo "STORAGE_CLASS=$(STORAGE_CLASS)" > config/resources/storage.env
	oc apply -k config/resources
	$(MINIO_ROLLOUT)
	$(LOGGING_ROLLOUT)
	$(NETOBSERV_ROLLOUT)

clean-operators:
	oc delete --ignore-not-found -k config/operators
clean-resources:
	oc delete --ignore-not-found -k config/resources
clean: clean-resources clean-operators
