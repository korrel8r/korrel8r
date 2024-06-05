# Deploy operators and resources for observability components.

all: operators resources

operators:
	kubectl apply -k config/operators
	../wait.sh subscription openshift-logging cluster-logging
	../wait.sh subscription openshift-operators-redhat loki-operator
	../wait.sh subscription openshift-netobserv-operator netobserv-operator
	kubectl patch consoles.operator.openshift.io/cluster --type=merge --patch '{ "spec": { "plugins": ["monitoring-plugin", "logging-view-plugin", "netobserv-plugin"]}}'

STORAGE_CLASS?=$(shell kubectl get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}')

MINIO_ROLLOUT=../wait.sh rollout minio deployment/minio

LOGGING_ROLLOUT=../wait.sh rollout openshift-logging \
deployment.apps/cluster-logging-operator \
deployment.apps/logging-loki-distributor \
deployment.apps/logging-loki-gateway \
deployment.apps/logging-loki-querier \
deployment.apps/logging-loki-query-frontend \
deployment.apps/logging-view-plugin


NETOBSERV_ROLLOUT=../wait.sh rollout netobserv \
deployment.apps/loki-distributor \
deployment.apps/loki-gateway \
deployment.apps/loki-querier \
deployment.apps/loki-query-frontend \
deployment.apps/netobserv-plugin

resources: config/resources/storage.env
ifeq ($(strip $(STORAGE_CLASS)),)
		$(error cannot determine storage class of cluster. exitting)
endif
	kubectl apply -k config/resources
	$(MINIO_ROLLOUT)
	$(LOGGING_ROLLOUT)
	$(NETOBSERV_ROLLOUT)

clean-operators:
	kubectl delete --ignore-not-found -k config/operators
	kubectl patch consoles.operator.openshift.io/cluster --type=merge --patch '{ "spec": { "plugins": [] }}'
clean-resources: config/resources/storage.env
	kubectl delete --ignore-not-found -k config/resources
clean-all: clean-resources clean-operators

config/resources/storage.env: $(MAKEFILE_LIST)
	echo "STORAGE_CLASS=$(STORAGE_CLASS)" > $@
