#!/bin/bash
# Run MCP server for korrel8r, default to COO route.

KORREL8R_URL=${KORREL8R_URL:-$(oc get routes/korrel8r -n openshift-cluster-observability-operator -o template='https://{{.spec.host}}/api/v1alpha1')} || \
  { echo "Cannot find KORREL8R_URL"; exit 1; }

cd $(dirname $0)

proxy() {
  # Set openapi-mcp parameters via environment variables.
  export BEARER_TOKEN=$(oc whoami -t)
  openapi-mcp --base-url $KORREL8R_URL --log-file $0.log "$@" ../korrel8r-openapi.yaml
}

builtin() {
  go run ../../cmd/korrel8r -v3 -c ../../etc/korrel8r/openshift-route.yaml mcp 2> /tmp/k8r.log
}

builtin
