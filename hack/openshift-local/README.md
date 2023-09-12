# Set up Openshift Local for Korrel8r

## Create your cluster

[Install Openshift Local](https://developers.redhat.com/products/openshift-local/overview) (formerly known as Code Ready Containers or CRC)

## Installingn Logging

### Install operators

1. From OperatorHub. Using the Openshift Console, install with defaults:
   - "Red Hat Openshift Logging"
   - "Loki operator"

1. From source repositories. \
   **NOTE**: _The steps below may be out of date, check the repository README for the latest instructions_
   - Cluster logging https://github.com/openshift/cluster-logging-operator
   ```
   oc create ns openshift-logging
   make deploy-image deploy-catalog install
   ```
   - LokiStack store https://github.com/grafana/loki/tree/main/operator
   ```
   oc create ns openshift-operators-redhat
   make VARIANT=openshift REGISTRY_BASE=quay.io/alanconway VERSION=v0.0.2-test olm-undeploy olm-deploy
   ```

### Create resources

    make logging

This will create resources in the `openshift-logging` namespace:

1.  An extra-small LokiStack deployment for log storage and query.
1.  A CluserLogging instance using vector to forward to LokiStack.
1.  A ClusterLogForwarder instance to forward all logs (including audit logs)

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


