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

This project is an experimental *correlation engine* which applies a set of *rules* to a "start" signal,
and produces a query for a set of related "goal" signals.

For example: Given an Alert, I want to see related Logs:

1. The Alert refers to a PVC.
2. The K8s API server is queried for all the Pods mounting that PVC.
3. The identity of the Pods is used to create a query for associated logs around the time of the Alert.
4. The log store is queried and returns relevant log data.


For more details see the [Go API documentation](https://pkg.go.dev/github.com/alanconway/korrel8/)

## Key Concepts

- **Domain**: a family of signals with common storage and representation. \
  Examples: resource, alert, metric, trace
- **Store**: a source of signal data from some \
  Examples: Loki, Prometheus, Kubernetes API server.
- **Query**: Stores accept a Query and return a set of matching signals.
- **Class**: A subset of signals in a Domain with a common same schema: field names, field types and semantics. \
  Examples: Pod (k8s), Event(k8s), KubeContainerWaiting(alert), log_logged_bytes_total(metric)
- **Object**: An instance of a signal. 

## Object and Rules

All objects can marshal as JSON or YAML.
Each domain may defines its own object types, with their own JSON encoding.
For example the k8s domain will decode JSON into Go API objects, with typed fields converted from JSON.

Rules have a start Class and a goal Class. Rules take an Object and generate a Query.
Rules are written in terms of the specific domain objects, field names and query languages they deal with.

Existing rules are defined using Go templates, see ./rules for examples.


