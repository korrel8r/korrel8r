# Deploy operators and resources for observability components.

help:
	Use: make [operators|resources|clean-operators|clean-resources]

operators:
	kubectl apply -k config/operators
	../wait.sh subscription -n openshift-logging cluster-logging
	../wait.sh subscription -n openshift-operators-redhat loki-operator
	../wait.sh subscription -n openshift-netobserv-operator netobserv-operator
	kubectl patch consoles.operator.openshift.io/cluster --type=merge --patch '{ "spec": { "plugins": ["monitoring-plugin", "logging-view-plugin", "netobserv-plugin"]}}'

STORAGE_CLASS?=$(shell kubectl get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}')

MINIO_ROLLOUT=../wait.sh rollout -n minio deployment/minio

LOGGING_ROLLOUT=../wait.sh rollout -n openshift-logging	\
deployment.apps/cluster-logging-operator		\
deployment.apps/logging-loki-distributor		\
deployment.apps/logging-loki-gateway			\
deployment.apps/logging-loki-querier			\
deployment.apps/logging-loki-query-frontend		\
deployment.apps/logging-view-plugin


NETOBSERV_ROLLOUT=../wait.sh rollout -n netobserv	\
deployment.apps/loki-distributor			\
deployment.apps/loki-gateway				\
deployment.apps/loki-querier				\
deployment.apps/loki-query-frontend			\
deployment.apps/netobserv-plugin

resources:
	echo "STORAGE_CLASS=$(STORAGE_CLASS)" > config/resources/storage.env
	kubectl apply -k config/resources
	$(MINIO_ROLLOUT)
	$(LOGGING_ROLLOUT)
	$(NETOBSERV_ROLLOUT)

clean-operators:
	kubectl delete --ignore-not-found -k config/operators
clean-resources:
	kubectl delete --ignore-not-found -k config/resources
clean-all: clean-resources clean-operators
