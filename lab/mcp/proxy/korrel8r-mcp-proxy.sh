#!/bin/bash
# Run a MCP stdio server that forwards to the korrel8r REST API.

# Default URL is for the korrel8r instance deployed by COO in the cluster.
KORREL8R_URL=${KORREL8R_URL:-$(oc get routes/korrel8r -n openshift-cluster-observability-operator -o template='https://{{.spec.host}}/api/v1alpha1')}

# The proxy uses the OpenAPI spec to translate MCP calls to REST calls.
SPEC=${SPEC:-$(dirname $0)/../korrel8r-openapi.yaml}
LOG=${LOG:-/tmp/openapi-mcp.log}

export BEARER_TOKEN=$(oc whoami -t)
exec openapi-mcp --base-url $KORREL8R_URL --log-file $LOG "$@" $SPEC
