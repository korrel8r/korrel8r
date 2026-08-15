#!/bin/bash
# Run benchmark scenarios with metrics dump.

korrel8r -c testdata/use_mock_store.yaml neighbors -q 'k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}' --metric-file neighbors-metrics.json -o json-pretty > neighbors-out.json
korrel8r -c testdata/use_mock_store.yaml goals -q 'k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}' "k8s:Service.v1" "log:infrastructure" --metric-file goals-metrics.json -o json-pretty > goals-out.json
