NAMESPACES= -f manifests/0namespaces.yaml
WATCH=oc get events -A --watch-only& trap "kill %%" EXIT;

all: operators logging

logging:
	oc apply -f ./manifests/
	$(WATCH) until oc get -n openshift-logging $(LOGGGING_DEPLOYMENTS) $(LOGGING_STATEFULSETS); do echo waiting...; sleep 3;  done
	$(WATCH) oc wait -n openshift-logging --for=condition=available $(LOGGING_DEPLOYMENTS)
	$(WATCH) for S in $(LOGGING_STATEFULSETS); do echo $$S; oc rollout status  -n openshift-logging --watch $$S; done

operators:
	oc apply -f ./manifests/0namespaces.yaml
	oc apply -f manifests/operators/
	$(WATCH) until oc get -n openshift-logging $(LOGGING_CSVS); do echo waiting...; sleep 3;  done
	$(WATCH) oc wait --for=jsonpath='{.status.phase}'=Succeeded -n openshift-logging $(LOGGING_CSVS)

namespaces:
	oc apply $(NAMESPACES)

delete:
	oc delete --ignore-not-found $(LOGGING_MANIFESTS) --now || true

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

LOGGING_CSVS=									\
	clusterserviceversion.operators.coreos.com/cluster-logging.v5.8.0	\
	clusterserviceversion.operators.coreos.com/loki-operator.v5.8.0

delete-all: delete # Delete resources and operators
	oc delete --ignore-not-found subscription,replicaset,deployment,service -n openshift-operators-redhat -l app.kubernetes.io/part-of=cluster-logging
	oc delete --ignore-not-found $(NAMESPACES)
	oc delete --ignore-not-found operator/cluster-logging.openshift-logging operator/loki-operator.kubernetes-operators
	oc delete --ignore-not-found -A crd,clusterrole -l app.kubernetes.io/part-of=cluster-logging
