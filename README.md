**⚠ Experimental ⚠** This code may change or vanish. It may not work. It may not even make sense.

# Overview

A Kubernetes cluster generates many types of *observable signal*, including:

|                   |                                                                                                                          |
|-------------------|--------------------------------------------------------------------------------------------------------------------------|
| Logs              | Application, infrastructure and audit logs from Pods and cluster nodes.                                                  |
| Metrics           | Counts and measurements of system behavior.                                                                              |
| Alerts            | Rules that fire when metrics cross important thresholds.                                                                 |
| Traces            | Nested execution spans describing distributed requests.                                                                  |
| Kubernetes Events | Describe significant events in a cluster.                                                                                |
| Network Events    | TCP and IP level network information.                                                                                    |
| Resources         | Spec and status information. Not traditionally considered signals, but often the starting point or goal for correlation. |

Different signal types use different vocabularies to identify the same things.
For example:

-   `k8s.pod.name` (traces)
-   `pod` or `pod_name` (metrics)
-   `kubernetes.pod_name` (logs)

This project is a *correlation engine* which applies *rules* to correlate signals of different types and vocabularies.
Given a starting point (e.g. an alert) and a goal (e.g. \`\`find logs'') the engine follows rules to find a set of goal signals that are related to the starting signal.

The engine encapsulates a common set of rules that can be used in many settings: graphical consoles, command line tools, offline data-processing and other services.

Goals include:

-   Encode domain knowledge from SREs and other experts as re-usable rules.
-   Automate navigation from symptoms to data that helps diagnose causes.
-   Reduce multiple-step manual procedures and ad-hoc scripts to a single click/query for the operator.
-   Help tools that gather and/or analyze diagnostic data to find and focus on useful data.

For more details:

-   [Go API documentation on pkg.go.dev](https://pkg.go.dev/github.com/korrel8/korrel8/)
-   [Source code on GitHub](https://github.com/korrel8/korrel8)

# Request for Input

If you work with OpenShift or kubernetes clusters, your experience can help to build a useful rule-base.
If you are interested, please [create a GitHub issue](https://github.com/korrel8/korrel8/issues/new), following this template:

## 1. When I am in this situation: ＿＿＿＿

Situations where:
- you have some information, and want to use it to jump to related information
- you know how get there, but it’s not trivial: you have to click many console screens, type many commands, write scripts or other automated tools.

The context could be
- interacting with a cluster via graphical console or command line.
- building services that will run in a cluster to collect or analyze data.
- out-of-cluster analysis of cluster data.

## 2. And I am looking at: ＿＿＿＿

Any type of signal or cluster data: metrics, traces, logs alerts, k8s events, k8s resources, network events, add your own…

The data could be viewed on a console, printed by command line tools, available from files or stores (loki, prometheus …)

## 3. I would like to see: ＿＿＿＿

Again types of information include: metrics, traces, logs alerts, k8s events, k8s resources, network events, add your own…

Describe the desired data, and the steps needed to get from the starting point in step 2.

Examples:
- I’m looking at this alert, and I want to see …
- I’m looking at this k8s Event, and I want to see …
- There are reports of slow responses from this Service, I want to see…
- CPU/Memory is getting scarce on this node, I want to see…
- These PVs are filling up, I want to see…
- Cluster is using more storage than I expected, I want to see…

# Implentation Concepts

The following concepts are represented by interfaces in the korrel8 package. Support for a new domain implements these interfaces:

**Domain** \
a family of signals with common storage and representation. Examples: resource, alert, metric, trace

**Store** \
a source of signal data from some + Examples: Loki, Prometheus, Kubernetes API server.

**Query**  \
Stores accept a Query and return a set of matching signals.

**Class**  \
A subset of signals in a Domain with a common same schema: field names, field types and semantics. + Examples: Pod (k8s), Event(k8s), `KubeContainerWaiting`(alert), log_logged_bytes_total(metric)

**Object** \
An instance of a signal.

**Rule**  \
Apply to an instance of a *start* Class, generate a query for the *goal* Class. + Rules are written in terms of domain-specific objects and query languages, but the start and goal can be in different domains (e.g. k8s/Service.v1 → loki/log) + Currently rules are defined as Go templates, see ./rules for examples.

# Conflicting Vocabularies

The correlation problem would be simpler if there was a single vocabulary to describe resource and signal attributes.
The [Open Telemetry Project](https://opentelemetry.io/) aims to create such a standard vocabulary for observability.

OpenShift tracing uses Open Telemetry notation. However, OpenShift logging and metrics do not.
They each use a different convention that was established before Open Telemetry existed.

Historically, observability tools have developed in "silos" without standardization.
Different conventions adopted in each domain are now entrenched and difficult to change.
A single vocabulary may eventually become universal, but in the medium term we have to handle mixed signals.
