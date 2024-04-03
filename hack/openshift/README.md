# Set up OpenShift for Korrel8r demos

<!--toc:start-->
- [Set up OpenShift for Korrel8r demos](#set-up-openshift-for-korrel8r-demos)
  - [Create your cluster](#create-your-cluster)
  - [Installing Operators and Logging resources](#installing-operators-and-logging-resources)
    - [View logs](#view-logs)
  - [Network Observability](#deploy-network-observability-operator)
     - [View Network Flows](#view-network-flows)
  - [Metrics, Alerts](#metrics-alerts)
  - [Events](#events)
  - [Uninstalling](#uninstalling)
<!--toc:end-->

These instructions will help you set up a small cluster with observable signals for test or demonstration purposes.
This is not intended for production clusters.

## Create your cluster

To get a personal test-cluster on your own machine,
[install OpenShift Local](https://developers.redhat.com/products/openshift-local/overview)
(formerly known as Code Ready Containers or CRC)

**NOTE**: These instructions will also work for other types of OpenShift clusters.
## Installing Operators and Logging resources

**Note**: You can also install the operators manually from OperatorHub:

1. Red Hat OpenShift Logging
1. Loki operators

```bash
make deploy
```

This will create resources in the `openshift-logging` namespace:

1. Deploy Red Hat OpenShift Logging and Loki Operators
1. An extra-small LokiStack deployment for log storage and query.
1. A CluserLogging instance using vector to forward to Loki Stack.
1. A ClusterLogForwarder instance to forward all logs (including audit logs)

### View logs

From the OpenShift console: Observe > Logs

## Deploy Network Observability Operator

- Create a namespace `oc create ns netobserv`
- Deploy a Loki Operator in the `netobserv` namespace
- Create a small Lokistack deployment for network flow storage. An example manifest file is provided.
- Customize the manifest to create a secret, update the namespace and storage class as appropritate to the cluster.
- Install Network Observability operator from Operator Hub using the documentation here - [Install Network Observailty operator](https://docs.openshift.com/container-platform/4.15/observability/network_observability/installing-operators.html#network-observability-operator-installation_network_observability)
- Create the `FlowCollector` CR and make sure you select the spec.loki.mode as `Lokistack`

### View Network Flows

From the OpenShift console: Observe > Network Flows

## Metrics, Alerts

Installed with OpenShift.

- OpenShift console: Observe

## Events

Built in to k8s.

- OpenShift console: Home > Events
- Command line: `oc get events`

## Uninstalling

```bash
make undeploy
```
