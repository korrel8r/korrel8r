# Deploy operators and resources for observability components.

all: operators resources

operators:
	kubectl apply -k config/operators
	../wait.sh subscription openshift-logging cluster-logging
	../wait.sh subscription openshift-operators-redhat loki-operator
	../wait.sh subscription openshift-netobserv-operator netobserv-operator
	kubectl patch consoles.operator.openshift.io/cluster --type=merge --patch '{ "spec": { "plugins": ["monitoring-plugin", "logging-view-plugin", "netobserv-plugin"]}}'

# Find default storage class.
STORAGE_CLASS?=$(strip $(shell kubectl get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}'))
ifeq ($(STORAGE_CLASS),)
$(error cannot determine storage class of cluster, set STORAGE_CLASS.)
endif

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

resources: $(STORAGE_ENV)
	kubectl apply -k config/resources
	$(MINIO_ROLLOUT)
	$(LOGGING_ROLLOUT)
	$(NETOBSERV_ROLLOUT)

clean-operators:
	kubectl delete --ignore-not-found -k config/operators
	kubectl patch consoles.operator.openshift.io/cluster --type=merge --patch '{ "spec": { "plugins": [] }}'

clean-resources:
	touch $(STORAGE_ENV)	# So kustomize won't choke on the missing file.
	kubectl delete --ignore-not-found -k config/resources
	rm $(STORAGE_ENV)

clean-all: clean-resources clean-operators

STORAGE_ENV=config/resources/storage.env
ifneq ($(STORAGE_CLASS),$(file < $(STORAGE_ENV)))
.PHONY: $(STORAGE_ENV)
endif
$(STORAGE_ENV):
	echo "STORAGE_CLASS=$(STORAGE_CLASS)" > $@
