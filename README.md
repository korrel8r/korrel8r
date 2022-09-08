# Correlating Observable Signals in a Kubernetes Cluster

A Kubernetes cluster generates many types of *observable signal*, including:

- **Logs**: application, infrastructure and audit logs from Pods and cluster nodes.
- **Metrics**: Counts and measurements of system behavior.
- **Alerts**: Rules that fire when metrics cross important thresholds.
- **Traces**: nested execution spans describing distributed requests.
- **Events**: events.k8s.io objects describe significant events in a cluster.
- **Resources**: Although not traditionally considered 'signals', cluster resources have status information that can be observed, and spec information that can be correlated with other signals.

This project aims to provide a "correlation engine" to automate the process of taking a "start" signal and producing a set of "goal" signals that are related to it.
The engine automatically follows relationships (expressed as Rules) to get to the goal.

For example: Given an Alert, I want to see related Logs.
1. The Alert refers to a PVC.
2. The k8s API server can be queried for all the Pods mounting that PVC.
3. The identity of those Pods can be used to form a query to get the associated logs.

The correlation engine will construct and follow such a chain of rules automatically.
This means the user spends less time manually constructing queries and following relationships,
and can get directly to looking at the signal data.

Frequently the different types of signal are represented by different data formats,
and are propagated, stored and viewed using separate tools.
Worse, they often use different "vocabularies" for labels that refer to the same things.
For example: A label for a Pod name may be `pod`, `podname`, `k8s.pod.name`, `kubernetes.pod_name`
depending on the type of signal carrying the label.
Correlation rules that cross domains manage the translation of different label vocabularies.

Packages:
- [korrel8](https://pkg.go.dev/github.com/alanconway/korrel8/pkg/korrel8) Generic interfaces and algorithms. Start here.
- [other packages](https://pkg.go.dev/github.com/alanconway/korrel8/pkg) Domain-specific implementations for the generic interfaces.

## TODO

- Automate discovery & generation of rules for known resource patterns?
