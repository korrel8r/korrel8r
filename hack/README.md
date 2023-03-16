This directory contains scripts and manifests to setup a metrics+logs stack on
a Kind cluster.

## Pre-requisites

The following tools need to be installed in your path:
* kubectl
* kind
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
* `http://prometheus.127.0.0.1.nip.io`
* `http://alertmanager.127.0.0.1.nip.io`
* `http://grafana.127.0.0.1.nip.io`
* `http://loki.127.0.0.1.nip.io`

## Using with korrel8r

```bash
korrel8r --alerts-url http://alertmanager.127.0.0.1.nip.io \
  --metrics-url http://prometheus.127.0.0.1.nip.io \
  --logs-url http://loki.127.0.0.1.nip.io ...
```

