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

Follow correlation paths from start objects to specific goal classes. Returns a graph of correlated classes with queries and result counts. Use for targeted queries like "find logs for this pod" or "what alerts fired for this deployment?" Start queries use "domain:class:selector" format; goals are class names like ["log:application"]. See 'help' for syntax.

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

Follow correlation rules outward from start objects up to a given depth. Returns a graph of correlated classes with queries and result counts. Use for open-ended exploration like "what is related to this pod?" Depth 1 = direct correlations; depth 2-3 typically reaches logs, metrics, and alerts. Start queries use "domain:class:selector" format; see 'help' for syntax.

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

Get what the user is looking at in the console. Returns a view query (main console view) and/or search parameters (troubleshooting panel), either may be absent. Use these as context for further actions.

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `search` | object |  | The troubleshooting panel displays the results of this correlation search. |
| `view` | string |  | Query for the main console view, in DOMAIN:CLASS:SELECTOR format. |

## get_objects

Execute a query and return matching objects as self-contained JSON (all labels/fields included per object). Query format is "domain:class:selector"; see 'help' for syntax. Use the constraint parameter (limit number of objects, start/end time as RFC 3339) to control result size, especially for high-volume domains like logs, metrics, and traces.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `constraint` | object |  | Optional constraint to limit results by time range and/or count. |
| `query` | string | yes | Query string in the form 'domain:class:selector'. Use 'help' to learn query syntax for each domain. |

## help

Get help on domains, classes, and query syntax. Omit the domain parameter for help on all domains. Class names have the form "domain:class". Query strings have the form "domain:class:selector". Use this before constructing queries for other tools.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domain` | string |  | If specified, get help for this domain only. |

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `documentation` | string | yes | Domain documentation including query syntax and examples |

## list_domain_classes

List classes in a domain. A class represents objects with a specific structure. Full class names have the form "domain:class" and are used in queries and as goal parameters. Use 'help' for details on class and query syntax.

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

List available domains with descriptions. A domain groups signals or resources that share a query syntax and data store. Use 'list_domain_classes' or 'help' to explore a domain further.

### Output parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `domains` | object[] | yes | List of domains |

## show_in_console

Update the console to display new data. Set 'view' to a query to update the main view, and/or set 'search' to display a correlation graph in the troubleshooting panel. See 'help' for query syntax.

### Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `search` | object |  | The troubleshooting panel displays the results of this correlation search. |
| `view` | string |  | Query for the main console view, in DOMAIN:CLASS:SELECTOR format. |

