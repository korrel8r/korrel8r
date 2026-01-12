#!/bin/bash
# Once-off script to generate a mock store (YAML file) from real stores.

ROOT=$(git rev-parse --show-toplevel)

neighbors() {  go run "$ROOT"/cmd/korrel8r neighbors -d4 -q "$1" -o json; }

{
  neighbors 'k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}'
  neighbors 'trace:span:{}'
} | tee fixme.txt | "$ROOT"/hack/graphs_to_queries.sh | "$ROOT"/hack/queries_to_store.sh > mock_store.yaml
