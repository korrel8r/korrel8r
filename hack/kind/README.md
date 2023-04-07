This directory contains scripts and manifests to setup a metrics+logs stack on
a Kind cluster.

## Pre-requisites

The following tools need to be installed in your path:
* kubectl
* kind (e.g. `go install sigs.k8s.io/kind@latest`)
* helm

## Getting started

* Create a kind cluster:

```bash
./setup.sh create test-cluster
```

* Verify that you can access the cluster's API with kubectl.

* Deploy the metrics+logs stack:

```bash
./setup.sh apply
```

The following services are accessible via ingress:
* `http://prometheus.127.0.0.1.nip.io:8000`
* `http://alertmanager.127.0.0.1.nip.io:8000`
* `http://grafana.127.0.0.1.nip.io:8000`
* `http://loki.127.0.0.1.nip.io:8000`
* `http://dashboard.127.0.0.1.nip.io:8000`

## Using with korrel8r

```bash
korrel8r --alerts-url http://alertmanager.127.0.0.1.nip.io:8000 \
  --metrics-url http://prometheus.127.0.0.1.nip.io:8000 \
  --logs-url http://loki.127.0.0.1.nip.io:8000 ...
```

## Updating the manifests

The files in the `manifests/` directory are generated from jsonnet (for
kube-prometheus manifests) and kustomize. To update the generated manifests,
run:

```bash
./generate.sh
```
