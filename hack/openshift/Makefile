WATCH=oc get events -A --watch-only& trap "kill %%" EXIT;
APPLY=$(WATCH) oc apply $(foreach NAME, $(1), -f manifests/$(NAME).yaml)

NAMESPACES= -f manifests/0namespaces.yaml
LOGGING= -f manifests/lokistack.yaml -f manifests/clusterlogging.yaml -f manifests/clusterlogforwarder.yaml -f manifests/chat.yaml

all: minio logging

logging:
	$(WATCH) oc apply $(NAMESPACES) $(LOGGING)
	$(MAKE) wait-ready || { echo; echo "Check for relevant workaround targets: "; grep workaround $(MAKEFILE_LIST); }

minio:
	$(WATCH) oc apply $(NAMESPACES) -f manifests/minio.yaml
	$(WATCH) oc wait --for=condition=available deployment.apps/minio -n openshift-logging


wait-ready:
	$(WATCH) until oc get -n openshift-logging $(DEPLOYMENTS); do echo waiting...; sleep 5;  done
	$(WATCH) oc wait -n openshift-logging --for=condition=available $(DEPLOYMENTS)
	$(WATCH) until oc get -n openshift-logging $(STATEFULSETS); do echo waiting...; sleep 5; done
	$(WATCH) for S in $(STATEFULSETS); do echo $$S; oc rollout status  -n openshift-logging --watch $$S; done

DEPLOYMENTS=						\
	deployment.apps/cluster-logging-operator	\
	deployment.apps/logging-loki-distributor	\
	deployment.apps/logging-loki-gateway		\
	deployment.apps/logging-loki-querier		\
	deployment.apps/logging-loki-query-frontend	\
	deployment.apps/logging-view-plugin

STATEFULSETS=						\
	statefulset.apps/logging-loki-compactor		\
	statefulset.apps/logging-loki-index-gateway	\
	statefulset.apps/logging-loki-ingester

delete:
	oc delete --ignore-not-found $(LOGGING) --wait=true || true

delete-all: delete # Delete resources first, then operators
	oc delete --ignore-not-found -f manifests/minio.yaml --wait
	oc delete --ignore-not-found subscription,replicaset,deployment,service -n openshift-operators-redhat -l app.kubernetes.io/part-of=cluster-logging
	oc delete --ignore-not-found $(NAMESPACES)
	oc delete --ignore-not-found operator/cluster-logging.openshift-logging operator/loki-operator.kubernetes-operators
	oc delete --ignore-not-found -A crd,clusterrole -l app.kubernetes.io/part-of=cluster-logging

