# Signal Correlation for Kubernetes and Beyond

**⚠ Warning: Experimental ⚠**: This code may change without warning.

[API documentation is available at pkg.go.dev](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/korrel8r)

## Quick Start ##

You need to be logged in to an openshift cluster as an admin for this to work

```bash
go install github.com/korrel8r/korrel8r/cmd/korrel8r
korrel8r web &
xdg-open http://localhost:8080
# Replace xdg-open with your preferred browser if it doesn't work on your system.
```

## Overview ##

Korrel8r is a *correlation engine* that follows relationships to find related data in multiple heterogeneous stores.

Korrel8r uses a set of *rules* that describe relationships between *objects* and *signals*. 
Given a *start* object (e.g. an Alert in a cluster) and a *goal* (e.g. "find related logs") the engine searches 
for goal data that is related to the start object some chain of rules.

The set of rules captures expert knowledge about troubleshooting in an executable form.
The engine aims to provide common rule-base that can be re-used in many settings:
as a service, embedded in graphical consoles or command line tools, or in offline data-processing systems.

The goals of this project include:

- Encode domain knowledge from SREs and other experts as re-usable rules.
- Automate navigation from symptoms to data that helps diagnose causes.
- Reduce multiple-step manual procedures to fewer clicks or queries.
- Help tools that gather and analyze diagnostic data to focus on relevant information.
- Bring together data that is held in different types of store.

## Signals and Objects ##

A Kubernetes cluster generates many types of *observable signal*, including:

| Signal Type       | Description                                                             |
|-------------------|-------------------------------------------------------------------------|
| Metrics           | Counts and measurements of system behaviour.                            |
| Alerts            | Rules that fire when metrics cross important thresholds.                |
| Logs              | Application, infrastructure and audit logs from Pods and cluster nodes. |
| Kubernetes Events | Describe significant events in a cluster.                               |
| Traces            | Nested execution spans describing distributed requests.                 |
| Network Events    | TCP and IP level network information.                                   |

A cluster also contains objects that are not usually considered "signals",
but which can be correlated with signals and other objects:

| Object Type   | Description                                    |
|---------------|------------------------------------------------|
| k8s resources | Spec and status information.                   |
| Run books     | Problem solving guides associated with Alerts. |
| k8s probes    | Information about resource state.              |
| Operators     | Operators control other resources.             |

Korrel8r uses the term "object" generically to refer to signals and objects.

## Implentation Concepts ##

The following concepts are represented by interfaces in the korrel8r package.
These interfaces are implemented for each distinct type of signal and store.

**Domain** \
A family of signals or objects with common storage and representation.
Examples: k8s (resource), alert, metric, log, trace

**Store** \
A source of signal data from some Domain.
Examples: Loki, Prometheus, Kubernetes API server.

**Query**  \
A Query selects a set of signals from a store.
Queries are expressed as JSON objects and generated by rule templates.
The fields and values in a query depend on the type of store it will be used with.

**Class**  \
A subset of signals in a Domain with a common schema (the same field definitions).
Examples: `k8s/Pod`, `logs/audit`

**Object** \
An instance of a signal or other correlation object.

**Rule**  \
A Rule applies to an instance of a *start* Class, and generates queries for a *goal* Class.
Rules are written in terms of domain-specific objects and query languages.
The start and goal of a rule can be in different domains (e.g. k8s/Pod → log)
Rules are defined using Go templates, see ./rules for examples.

## Conflicting Vocabularies ##

Different signal and object domains may use different vocabularies to identify the same things.
For example:

- `k8s.pod.name` (traces)
- `pod` or `pod_name` (metrics)
- `kubernetes.pod_name` (logs)

The correlation problem would be simpler if there was a single vocabulary to describe signal attributes.
The [Open Telemetry Project](https://opentelemetry.io/) aims to create such a standard vocabulary.
Unfortunately, at least for now, multiple vocabularies are embedded in existing systems.

A single vocabulary may eventually become universal, but in the short to medium term we have to handle mixed signals.
Korrel8r expresses rules in the native vocabulary of each domain, but allows rules to cross domains.

## Request for Feedback ##

If you work with OpenShift or kubernetes clusters, your experience can help to build a useful rule-base.
If you are interested, please [create a GitHub issue](https://github.com/korrel8r/korrel8r/issues/new?assignees=&labels=enhancement&template=feature_request.md&title=%5BRFE%5D).
