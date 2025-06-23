#!/bin/bash
# Run MCP server for korrel8r.
# Default to the korrel8r route in the COO namespace.
HOST=${HOST:-$(oc get routes/korrel8r -n openshift-cluster-observability-operator -o template='{{.spec.host}}')}

# Set openapi-mcp parameters via environment variables.
export BEARER_TOKEN=$(oc whoami -t)
exec openapi-mcp -base-url https://$HOST/api/v1alpha1 $(dirname $0)/../korrel8r-openapi.yaml
