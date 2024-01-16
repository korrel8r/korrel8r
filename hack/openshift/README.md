# Set up Openshift for Korrel8r demos

These instructions will help you set up a small cluster with observable signals for _test or demonstration_ purposes.
This is not intended for production clusters.

## Create your cluster

To get a personal test-cluster on your own machine,
[install Openshift Local](https://developers.redhat.com/products/openshift-local/overview)
(formerly known as Code Ready Containers or CRC)

**NOTE**: These instructions will also work for other types of  Openshift cluster, provided you
edit `manifests/lokistack.yaml` and set `storageClassName` to a storage class available on your cluster.
To see available storage classes:

``` shell
oc get storageclass
```

## Installing Logging

### Install operators

    make operators

**Note**: You can also install the operators manually from OperatorHub:
- "Red Hat Openshift Logging"
- "Loki operator"

### Create logging resources

    make logging

This will create resources in the `openshift-logging` namespace:

1.  An extra-small LokiStack deployment for log storage and query.
1.  A CluserLogging instance using vector to forward to LokiStack.
1.  A ClusterLogForwarder instance to forward all logs (including audit logs)

### View logs

From the Openshift console: Observe > Logs

## Metrics, Alerts

Installed with openshift.
- Openshift console: Observe

## Events

Built in to k8s.
- Openshift console: Home > Events
- Command line: `oc get events`


