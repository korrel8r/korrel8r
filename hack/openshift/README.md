# Configuring Openshift for Observability Experiments

## Pre-requisites

- Administrative login to an Openshift cluster with a storage class.
- Openshift Client `oc` installed in your path.

## Logs

### Install operators

Install options:

1. From OperatorHub via Openshift web Console
   - Install: "Red Hat Openshift Logging", "Loki operator"
   - Check "Enable Operator recommended cluster monitoring on this Namespace" each time
   - Accept all other defaults.
1. From source repositories.
   _Read repository instructions, the example steps below may change._
   - Cluster logging https://github.com/openshift/cluster-logging-operator
   - LokiStack store https://github.com/grafana/loki/tree/main/operator
   ```
   oc create ns openshift-operators-redhat
   make VARIANT=openshift REGISTRY_BASE=quay.io/alanconway VERSION=v0.0.2-test olm-undeploy olm-deploy
   ```

### Edit resources

Edit logging/lokistack.yaml field "storageClassName" to a storage class in your cluster.

``` shell
oc get storageclass
```

### Create resources

    make all

This will create resources in the `openshift-logging` namespace:

1.  A minio deployment to provide S3 storage back-end for LokiStack.
2.  An extra-small LokiStack deployment for log storage and query.
3.  A CluserLogging instance using vector to forward to LokiStack.
4.  A ClusterLogForwarder instance to forward all logs (by default audit logs are not forwarded)

### View logs

Openshift console: Observe > Logs

## Metrics, Alerts

Installed with openshift.
- Openshift console: Observe

## Events

Built in to k8s.
- Openshift console: Home > Events
- Command line: `oc get events`

## Traces

TBD


# Command line and programmatic access to signals

See [signal_samples](./signal_samples/README.md)
