---
title: MCP API
description: MCP tool reference
weight: 90
---
<!-- Generated content, do not edit! -->
Korrel8r provides an [MCP](https://modelcontextprotocol.io/) server with the following tools.

- [create_goals_graph](#create_goals_graph)
- [create_neighbors_graph](#create_neighbors_graph)
- [get_console](#get_console)
- [get_objects](#get_objects)
- [help](#help)
- [list_domain_classes](#list_domain_classes)
- [list_domains](#list_domains)
- [show_in_console](#show_in_console)

## create_goals_graph

Find a specific class of data related to a resource or signal, such as application logs for a Deployment. Use for targeted investigations: {"start":{"queries":["k8s:Deployment.v1.apps:{\"namespace\":\"default\",\"name\":\"my-deployment\"}"]},"goals":["log:application"]}. The selector is JSON text inside the query string. For unfamiliar classes or selectors, use 'list_domains', 'list_domain_classes', and 'help' with a domain. Application logs work via the Kubernetes API without a separate log store; Events are k8s resources (k8s:Event.v1), not a separate domain. Verify graph queries with 'get_objects'.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `goals` | string[] | yes | Goal classes in DOMAIN:CLASS format, e.g. log:application, alert:alert. |
| `start` | object | yes | Starting point for the search. |

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `edges` | object[] |  | List of graph edges. |
| `nodes` | object[] |  | List of graph nodes. |
| `truncation` | object |  |  |

## create_neighbors_graph

Explore what is related to a failing workload or other resource or signal. Use first for open-ended investigations across Kubernetes resources and observability signals instead of guessing related object names. Example: {"start":{"queries":["k8s:Deployment.v1.apps:{\"namespace\":\"default\",\"name\":\"my-deployment\"}"]},"depth":2}. The selector is JSON text inside the query string; depth 1 finds direct neighbors, 2 follows one more hop. Verify relevant graph queries with 'get_objects'; correlation is not proof of cause.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `depth` | integer | yes | Maximum number of correlation steps to follow from the start. Depth 1 returns direct correlations only. |
| `start` | object | yes | Starting point for the search. |

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `edges` | object[] |  | List of graph edges. |
| `nodes` | object[] |  | List of graph nodes. |
| `truncation` | object |  |  |

## get_console

Read the user's OpenShift console view/search when asked "what am I looking at?". Only works when the user is logged in to the OpenShift console and has enabled AI navigation for their session. Report if the console is unavailable.

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `search` | object |  | The troubleshooting panel displays the results of this correlation search. |
| `view` | string |  | Query for the main console view, in DOMAIN:CLASS:SELECTOR format. |

## get_objects

Fetch full objects to verify graph results. Pass a graph node's queries[].query (domain:class:selector), or a known query. Use 'help' with a domain if constructing an unfamiliar query. Bound results with constraint.limit or constraint.start/end (RFC 3339).

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `constraint` | object |  | Set limit to bound results; start/end (RFC 3339) for time ranges. |
| `query` | string | yes | Query string 'domain:class:selector'. Use 'help' for unfamiliar query syntax. |

## help

Use this when you need query syntax or examples for a specified domain.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domain` | string | yes | Name of the domain to get help for. |

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `documentation` | string | yes | Domain documentation including query syntax and examples |

## list_domain_classes

Use this when you need a domain's classes for queries or graph goals (domain:class). Use 'help' for unfamiliar selectors.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domain` | string | yes | Name of the domain to list |

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `classes` | string[] | yes | List of classes in the domain |
| `domain` | string | yes | Domain name |

## list_domains

Use this when you need to find known domains and configured stores. Use 'list_domain_classes' to find classes for a domain.

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domains` | object[] | yes | Known domains and configured stores. |

## show_in_console

Display a query (view) and/or graph (search) in the user's OpenShift console. Only works when the user is logged in to the OpenShift console and has enabled AI navigation for their session. Report if the console is unavailable.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `search` | object |  | The troubleshooting panel displays the results of this correlation search. |
| `view` | string |  | Query for the main console view, in DOMAIN:CLASS:SELECTOR format. |

