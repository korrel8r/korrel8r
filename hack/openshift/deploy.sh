#!/usr/bin/env bash

set -eu -o pipefail

# constants
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r MANIFESTS_DIR="$PROJECT_ROOT/hack/openshift/manifests"
declare -r LOGGING_DEPLOYMENTS=(
	cluster-logging-operator
	logging-loki-distributor
	logging-loki-gateway
	logging-loki-querier
	logging-loki-query-frontend
	logging-view-plugin
	minio
)
declare -r LOGGING_STATEFULSETS=(
	logging-loki-compactor
	logging-loki-index-gateway
	logging-loki-ingester
)
declare -r LOGGING_CSVS=(
	cluster-logging
	loki-operator
)
declare -r LOGGING_NS="openshift-logging"
declare -r LOKI_NS="openshift-operators-redhat"
declare -r CLUSTER_LOGGING="instance"
declare -r LOKI_STACK="lokistack-loki"
declare LOGS_DIR="$PROJECT_ROOT/tmp/deploy"
# declare -r LOCAL_BIN="$PROJECT_ROOT/tmp/bin"
declare -r LOKI_OPERATOR_DEPLOY_NAME="loki-operator-controller-manager"
declare -r CLUSTER_LOGGING_OPERATOR_DEPLOY_NAME="cluster-logging-operator"
declare SHOW_HELP=false
declare DELETE_RESOURCE=false
# config
declare STORAGE_CLASS=${STORAGE_CLASS:-""}

init_storage_class() {
	echo "getting storage class from cluster"
	STORAGE_CLASS=$(oc get storageclass -o=jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}')
}

init_logs_dir() {
	rm -rf "$LOGS_DIR-prev"
	mv "$LOGS_DIR" "$LOGS_DIR-prev" || true
	mkdir -p "$LOGS_DIR"
}

parse_args() {
	### while there are args parse them
	while [[ -n "${1+xxx}" ]]; do
		case $1 in
		--help | -h)
			shift
			SHOW_HELP=true
			return 0
			;;
		--delete)
			DELETE_RESOURCE=true
			return 0
			;; # exit the loop
		*)
			return 1
			;; # show usage on everything else
		esac
	done
	return 0
}
print_usage() {
	local scr
	scr="$(basename "$0")"

	read -r -d '' help <<-EOF_HELP || true
		Usage:
		  $scr
		  $scr  --delete
		 ─────────────────────────────────────────────────────────────────

		Options:
		  --delete                deletes all the resources
	EOF_HELP

	echo -e "$help"
	return 0
}

must_gather() {
	log_events "$LOGGING_NS"
	log_events "$LOKI_NS"
	watch_operator_errors
}

log_events() {
	local ns="$1"
	shift
	oc get events \
		-o custom-columns=FirstSeen:.firstTimestamp,LastSeen:.lastTimestamp,Count:.count,From:.source.component,Type:.type,Reason:.reason,Message:.message \
		-n "$ns" | tee "$LOGS_DIR/$ns-events.log"
}

watch_operator_errors() {
	oc logs -n "$LOGGING_NS" "deploy/$CLUSTER_LOGGING_OPERATOR_DEPLOY_NAME" | grep -i error | tee "$LOGS_DIR/cluster-logging-operator.log"
	oc logs -n "$LOKI_NS" "deploy/$LOKI_OPERATOR_DEPLOY_NAME" | grep -i error | tee "$LOGS_DIR/loki-operator.log"
}

deploy_operators() {
	echo "installing Red Hat OpenShift Logging & Loki Operator"

	echo "creating required namespace for operators to install"
	oc apply -f "$MANIFESTS_DIR/0namespaces.yaml" || {
		echo "failed to configure the required namespace"
		return 1
	}

	oc apply -f "$MANIFESTS_DIR/operators/" || {
		echo "failed to configure operators"
		return 1
	}
	for val in "${LOGGING_CSVS[@]}"; do
		echo "waiting for $val to be created"
		until csv=$(oc get -n "$LOGGING_NS" csv -o name | grep "$val"); do
			echo "sleeping for 5s"
			sleep 5
		done
		echo "waiting for $val to be succeeded"
		oc wait --for=jsonpath='{.status.phase}'=Succeeded -n "$LOGGING_NS" "$csv" --timeout=5m || {
			echo "$csv status is invalid"
			return 1
		}
	done

	echo "enabling logging console plugin"
	oc patch consoles.operator.openshift.io cluster --type=merge \
		--patch '{ "spec": { "plugins": ["logging-view-plugin"]}}' || {
		echo "failed to patch the cluster for enabling logging console"
		return 1
	}
	return 0
}

deploy_logging() {
	echo "deploying lokistack"
	oc process -p STORAGE_CLASS="$STORAGE_CLASS" -f "$MANIFESTS_DIR/lokistack.yaml" | oc apply -f - || {
		echo "failed to deploy lokistack"
		return 1
	}
	echo "deploying cluster logging"
	oc apply -f "$MANIFESTS_DIR/clusterlogging.yaml" || {
		echo "failed to deploy cluster logging"
		return 1
	}
	oc patch clusterlogging "$CLUSTER_LOGGING" --type=merge \
		--patch "{\"metadata\": {\"annotations\": {\"logging.openshift.io/ocp-console-migration-target\": \"$LOKI_STACK\"}}}" \
		-n "$LOGGING_NS" || {
		echo "failed to add annotation to clusterlogging CR"
		return 1
	}

	echo "deploying cluster log forwarder"
	oc apply -f "$MANIFESTS_DIR/clusterlogforwarder.yaml" || {
		echo "failed to deploy cluster log forwarder"
		return 1
	}
	return 0
}

validate() {
	echo "checking status"
	for deployment in "${LOGGING_DEPLOYMENTS[@]}"; do
		echo "checking status of $deployment deployment"
		oc rollout status -n "$LOGGING_NS" deployment "$deployment" --timeout=5m --watch || {
			echo "invalid status for $deployment"
			return 1
		}
	done
	for statefulset in "${LOGGING_STATEFULSETS[@]}"; do
		echo "checking status of $statefulset statefulset"
		oc rollout status -n "$LOGGING_NS" statefulset "$statefulset" --timeout=5m --watch || {
			echo "invalid status for $statefulset"
			return 1
		}
	done
	echo "all deployments and statefulsets are healthy"
	return 0
}

deploy_minio() {
	echo "deploying minio for lokistack"
	oc apply -f "$MANIFESTS_DIR/minio.yaml" || {
		echo "failed to deploy minio"
		return 1
	}
	return 0
}

deploy_app() {
	echo "deploying a dummy application for testing"
	oc apply -f "$MANIFESTS_DIR/chat.yaml" || return 1
	return 0
}

cleanup() {
	local label="app.kubernetes.io/part-of=cluster-logging"
	echo "cleaning up"
	oc delete clusterloggings -A --all || true
	oc delete clusterlogforwarder -A --all || true
	oc delete lokistack -A --all || true
	oc delete -f "$MANIFESTS_DIR/minio.yaml" || true
	oc delete -f "$MANIFESTS_DIR/chat.yaml" || true
	oc delete crd,clusterrole,clusterrolebinding -l "$label" -A || true
	oc delete operators cluster-logging.openshift-logging || true
	oc delete operators loki-operator.openshift-operators-redhat || true
	oc delete -f "$MANIFESTS_DIR/0namespaces.yaml" || true

	echo "resources and operators has been successfully uninstalled"
}

main() {
	parse_args "$@" || {
		print_usage
		echo "failed to parse args"
		return 1
	}
	$SHOW_HELP && {
		print_usage
		return 0
	}
	$DELETE_RESOURCE && {
		cleanup
		return 0
	}
	init_logs_dir
	init_storage_class
	deploy_operators || {
		echo "operators deployment failed"
		must_gather
		return 1
	}
	deploy_minio || {
		echo "minio deployment failed"
		must_gather
		return 1
	}

	deploy_logging || {
		echo "logging resources deployment failed"
		must_gather
		return 1
	}
	deploy_app || {
		echo "test app deployment failed"
		must_gather
		return 1
	}
	validate || {
		echo "resources deployment validation failed. check for above errors"
		must_gather
		return 1
	}

}
main "$@"
