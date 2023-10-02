NAMESPACES= -f manifests/0namespaces.yaml
WATCH=oc get events -A --watch-only& trap "kill %%" EXIT;

all: minio logging ## Deploy log collector and local Loki storage

logging:
	$(WATCH) oc apply $(NAMESPACES) $(LOGGING_MANIFESTS)
	$(WATCH) until oc get -n openshift-logging $(LOGGGING_DEPLOYMENTS) $(LOGGING_STATEFULSETS); do echo waiting...; sleep 3;  done
	$(WATCH) oc wait -n openshift-logging --for=condition=available $(LOGGING_DEPLOYMENTS)
	$(WATCH) for S in $(LOGGING_STATEFULSETS); do echo $$S; oc rollout status  -n openshift-logging --watch $$S; done

minio:
	$(WATCH) oc apply $(NAMESPACES) -f manifests/minio.yaml
	$(WATCH) oc wait --for=condition=available deployment.apps/minio -n openshift-logging


delete-logging:
	oc delete --ignore-not-found $(LOGGING_MANIFESTS) --now || true

LOGGING_MANIFESTS=				\
	-f manifests/lokistack.yaml		\
	-f manifests/clusterlogging.yaml	\
	-f manifests/clusterlogforwarder.yaml	\
	-f manifests/chat.yaml			\
	-f manifests/minio.yaml

LOGGING_DEPLOYMENTS=					\
	deployment.apps/cluster-logging-operator	\
	deployment.apps/logging-loki-distributor	\
	deployment.apps/logging-loki-gateway		\
	deployment.apps/logging-loki-querier		\
	deployment.apps/logging-loki-query-frontend	\
	deployment.apps/logging-view-plugin		\
	deployment.apps/minio

LOGGING_STATEFULSETS=					\
	statefulset.apps/logging-loki-compactor		\
	statefulset.apps/logging-loki-index-gateway	\
	statefulset.apps/logging-loki-ingester

delete-all: delete-logging # Delete resources and operators
	oc delete --ignore-not-found subscription,replicaset,deployment,service -n openshift-operators-redhat -l app.kubernetes.io/part-of=cluster-logging
	oc delete --ignore-not-found $(NAMESPACES)
	oc delete --ignore-not-found operator/cluster-logging.openshift-logging operator/loki-operator.kubernetes-operators
	oc delete --ignore-not-found -A crd,clusterrole -l app.kubernetes.io/part-of=cluster-logging
