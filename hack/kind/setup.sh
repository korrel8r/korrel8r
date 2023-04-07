#!/usr/bin/env bash
# Setup a kind cluster with all korrel8r signals
set -e -u -o pipefail

SCRIPT_PATH=$(readlink -f "$0")
SCRIPT_DIR=$(cd "$(dirname "$SCRIPT_PATH")" && pwd)

wait_ready() {
	kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=300s
}

create_kind() {
	kind create cluster --name "${CLUSTER_NAME:-korrel8r}" --config="${SCRIPT_DIR}/kind_config.yaml"
	wait_ready
}

create_ingress() {
	helm repo add traefik https://traefik.github.io/charts
	helm repo update
	kubectl create namespace traefik
	cat << EOF | helm install -f - --namespace traefik traefik traefik/traefik
image:
  name: traefik
  pullPolicy: IfNotPresent
service:
  type: NodePort
ports:
  web:
    expose: true
    nodePort: 30000
  websecure:
    expose: true
    nodePort: 30001
nodeSelector:
  ingress-ready: 'true'
EOF
	wait_ready
}

create_dashboard() {
       helm repo add kubernetes-dashboard https://kubernetes.github.io/dashboard/
       cat <<EOF | helm install dashboard kubernetes-dashboard/kubernetes-dashboard -f - -n kubernetes-dashboard --create-namespace
protocolHttp: true
service:
  externalPort: 8080
rbac:
  clusterReadOnlyRole: true
EOF
	wait_ready
}

apply() {
	kubectl apply --server-side --force-conflicts -f "${SCRIPT_DIR}/manifests/setup"
	kubectl apply --server-side --force-conflicts -f "${SCRIPT_DIR}/manifests/"
	wait_ready
}

case ${1:-"help"} in
create)
	create_kind
	create_ingress
	create_dashboard
	;;

apply)
	apply
	;;

help)
	echo "usage: $(basename "$0") { create | apply }"
	;;

esac
