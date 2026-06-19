---
title: MCP API
description: MCP tool reference
weight: 60
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

Search for correlations between start objects and specific goal classes.
Only follows paths from the start objects that lead to one of the specified goal classes.

Returns a graph where nodes represent classes (each with queries and result counts)
and edges represent correlation rules that were applied.

Use this for targeted investigation: "find logs related to this pod" or "what alerts fired for this deployment?"

The start parameter uses queries in "domain:class:selector" format.
Use 'help' to learn the class and query syntax for each domain.
Goals are full class names, e.g. ["log:application"], ["alert:alert", "metric:metric"].

Example: to find logs for a crashing pod, use:
  start: {"queries": ["k8s:Pod:{\"namespace\":\"myapp\",\"name\":\"web-0\"}"]}
  goals: ["log:application"]

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

## create_neighbors_graph

Search for correlated observability signals and resources starting from known objects.
Follows correlation rules outward from the start objects up to the specified depth.

Returns a graph where nodes represent classes (each with queries and result counts)
and edges represent correlation rules that were applied.

Use this for open-ended exploration: "what is related to this pod?" or "what resources are related to these traces?"

The start parameter requires queries in the format "domain:class:selector".
Use 'help' to learn the class and query syntax for each domain.
Depth controls how many correlation steps to follow (1 = direct correlations only).
Higher depths cast a wider net: depth 1 finds directly correlated objects,
depth 2-3 typically reaches related signals like logs, metrics, and alerts.

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

## get_console

If the user refers to a console, use this tool to find out what the user is looking at.
The result includes:
- view: a korrel8r query selecting data displayed in the main console view.
  Not set if the console is not displaying data.
- search: parameters for the correlation search displayed in the troubleshooting panel.
  Not set if the troubleshooting panel is not open.

Use view and search to understand what the user is looking at,
and include it as context for further planning or actions.

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `search` | object |  | The troubleshooting panel displays the results of this correlation search. |
| `view` | string |  | Query for the main console view, in DOMAIN:CLASS:SELECTOR format. |

## get_objects

Execute a query and return matching objects as complete JSON.
The query must be in the format "domain:class:selector".
Use 'help' to learn the query syntax for each domain.

The returned objects are self-contained: all relevant labels and fields are included in each object.
This differs from direct back-end APIs (e.g. Loki, Tempo) which use a compact "stream" format
where common labels are sent once per stream. The complete format is more verbose
but each object can be processed independently.

Use the optional constraint parameter to control result size:
- limit: maximum number of objects to return.
- start/end: time range (RFC 3339) to restrict results by timestamp.
Use constraints to avoid excessively large results, especially for
high-volume domains like logs, metrics, and traces.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `constraint` | object |  | Optional constraint to limit results by time range and/or count. |
| `query` | string | yes | Query string in the form 'domain:class:selector'. Use 'help' to learn query syntax for each domain. |

## help

Get help about korrel8r domains, classes, and query syntax.
Omitting the domain parameter returns help about all domains.

Class strings have the form "domain:class", where the legal values of "class" depend on the domain.

Query strings have the form "domain:class:selector".
The "domain:class" part indicates the class of data returned by the query.
The "selector" part is a domain-specific query string.

Use this tool to learn how to construct valid class names and queries for a domain before using tools that have class or query parameters.
For example: create_neighbors_graph, create_goals_graph, get_console or show_in_console.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domain` | string |  | If specified, get help for this domain only. |

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `documentation` | string | yes | Domain documentation including query syntax and examples |

## list_domain_classes

List the classes in a domain.
A class represents objects with a specific structure within a domain.
Some domains have a single class (e.g. metric:metric), others like k8s have many classes.
Use 'help' to get more details about a domain and its classes and queries.

Class names are used in queries and as goal parameters. The full class name is "domain:class".

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

Returns a list of Korrel8r domains with descriptions.
A domain contains observable signals or resources that use the same query syntax and data store.
Use this first to discover available domains, then use list_domain_classes to explore a domain.

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domains` | object[] | yes | List of domains |

## show_in_console

If the user refers to a console, use this tool to update the console to display new data.

- view: setting this field to a query updates the main view of the console to display the results of the query.
- search: setting this field displays a correlation graph in the console troubleshooting panel.

Use 'help' to learn the class and query syntax for each domain.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `search` | object |  | The troubleshooting panel displays the results of this correlation search. |
| `view` | string |  | Query for the main console view, in DOMAIN:CLASS:SELECTOR format. |

