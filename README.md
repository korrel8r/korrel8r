# Correlating Signals in a Kubernetes Cluster

> **⚠ Warning: Experimental!** *This code may change or vanish. It may not work. It may not even make sense.*

## Overview

A Kubernetes cluster generates many types of *observable signal*, including:

- **Logs**: Application, infrastructure and audit logs from Pods and cluster nodes.
- **Metrics**: Counts and measurements of system behavior.
- **Alerts**: Rules that fire when metrics cross important thresholds.
- **Traces**: Nested execution spans describing distributed requests.
- **Events**: Kubernetes `Event` objects describe significant events in a cluster.
- **Resources**: Not traditionally considered 'signals'; cluster resources have observable status and spec information, and are often the starting point for correlation.

This project is an experimental *correlation engine* to automate the process of taking a "start" signal and producing a set of "goal" signals that are related to it.
The engine automatically follows relationships (expressed as Rules) to get to the goal.

For example: Given an Alert, I want to see related Logs:

1. The Alert refers to a PVC.
2. The K8s API server is queried for all the Pods mounting that PVC.
3. The identity of the Pods is used to create a query for associated logs around the time of the Alert.
4. The log store is queried and returns relevant log data.

The correlation engine constructs and follows chains of rules like this automatically.
This means the cluster administrator can spend less time manually constructing queries and following relationships,
and can jump directly to looking at relevant signal data.

Frequently the different types of signal use different "vocabularies" to refer to the same things.
For example: A label for a Pod name may be called `pod`, `podname`, `k8s.pod.name`, `kubernetes.pod_name`
depending on the type of signal carrying the label.
The correlation engine translates between different label vocabularies.

Packages:
- [korrel8](https://pkg.go.dev/github.com/alanconway/korrel8/pkg/korrel8): Generic interfaces and algorithms. Start here.
- [other packages](https://pkg.go.dev/github.com/alanconway/korrel8/pkg): Domain-specific implementations.


## To-Do list

- [X] Path following and de-duplication.
- [X] Propagate time interval and other constraints on correlation.
- [X] Refactor Follower
- [X] Rename Result as Queries
- [X] Introduce Engine to bring together rules, domains and stores.
- [X] Serialize & deserialize Class, for command line.
- [ ] Constraint propagation.
- [ ] Query objects with exportable string forms.
- [ ] Multiple goals: avoid repeated queries (log + pod + metric...)
- [ ] "wildcard" classes: classes with common structure.
- [ ] Example correlation:
  - metric log_collector_error_count_total => logs + pods 
  - Traces?
- [ ] Wildcard Start() classes?
- [ ] Externalize (serialize) the rule base, allow it to be extended without recompile.
- [ ] Complete one sample correlation from alert to logs as demo.
- [ ] Query as value with fields - provide alternate serializations (e.g. URI or oc command, LogQL or URI)
  - Simplify k8s query, avoid URI parse & re-construct.
- [ ] Use streams as Object for prom & loki, more efficient & have same metadata. 

## Maybe later
- [ ] Use local loki executable instead of image to speed up tests?
