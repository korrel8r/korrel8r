---
title: REST API
description: HTTP API reference
---


HTTP API reference

## Table of Contents

HTTP Request | Description
-------------|------------
PUT [/config](#putconfig) | Change configuration settings at runtime.
GET [/domains](#getdomains) | Get the list of correlation domains.
GET [/domain/{domain}/classes](#getdomaindomainclasses) | Get the list of classes for a domain.
POST [/graphs/goals](#postgraphsgoals) | Create a correlation graph from start objects to goal queries.
POST [/graphs/neighbors](#postgraphsneighbors) | Create a neighborhood graph around a start object to a given depth.
POST [/graphs/neighbours](#postgraphsneighbours) | Create a neighborhood graph around a start object to a given depth.
POST [/lists/goals](#postlistsgoals) | Create a list of goal nodes related to a starting point.
GET [/objects](#getobjects) | Execute a query, returns a list of JSON objects.
PUT [/console](#putconsole) | Make console state available to an agent.
GET [/console/events](#getconsoleevents) | SSE event stream of console display updates from an agent.

## configure

### PUT /config {#putconfig}

Modify selected configuration settings (e.g. log verbosity) on a running service.


#### Query Parameters

- `verbose` *(integer)* Verbose level for logging.

### Responses

#### 200 Response

OK

```json
{}
```

#### Field Definitions

## console

### PUT /console {#putconsole}

Store console state so an agent can read it via MCP tool get_console. The MCP client must have the same session (Authorization header) as the REST client.


### Request

```json
{
   "search": {
      "goals": {
         "goals": [
            "k8s:Pod",
            "metric:metric"
         ],
         "start": {
            "class": {},
            "constraint": {
               "end": "2017-07-21T17:32:28.1341231Z",
               "limit": 1,
               "start": "2024-01-15T10:30:00Z"
            },
            "objects": [],
            "queries": [
               "k8s:Pod:{\"namespace\":\"default\",\"name\":\"my-pod\"}"
            ]
         }
      },
      "neighbors": {
         "depth": 95,
         "start": {
            "class": {},
            "constraint": {
               "end": "2017-07-21T17:32:28.1341231Z",
               "limit": 88,
               "start": "2024-01-15T10:30:00Z"
            },
            "objects": [],
            "queries": [
               "k8s:Pod:{\"namespace\":\"default\",\"name\":\"my-pod\"}"
            ]
         }
      }
   },
   "view": {}
}
```

#### Field Definitions

- `view` The main console view displays the results of this query.
- `search` The troubleshooting panel displays the results of this correlation search.

### Responses

#### 200 Response

Console display updated successfully

```json
{}
```

#### Field Definitions

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

### GET /console/events {#getconsoleevents}

Updates are triggered by update requests from MCP tool show_in_console. The MCP client must have the same session (Authorization header) as the REST client.


### Responses

#### 200 Response

SSE stream where each event's data field contains a JSON-encoded Console object.


## correlate

### POST /graphs/goals {#postgraphsgoals}

Specify a set of start objects, as queries or serialized objects, and a goal class. Returns a graph containing all paths leading from a start object to a goal object.


#### Query Parameters

- `options` *(object)* Options controlling the form of the returned graph.

### Request

```json
{
   "goals": [
      "k8s:Pod",
      "metric:metric"
   ],
   "start": {
      "class": {},
      "constraint": {
         "end": "2017-07-21T17:32:28.1341231Z",
         "limit": 12,
         "start": "2024-01-15T10:30:00Z"
      },
      "objects": [
         {}
      ],
      "queries": [
         "k8s:Pod:{\"namespace\":\"default\",\"name\":\"my-pod\"}"
      ]
   }
}
```

#### Field Definitions

- `goals` *(array of Class, required)* Goal classes in DOMAIN:CLASS format, e.g. log:application, alert:alert

- `start` Starting point for the search.

### Responses

#### 200 Response

OK

```json
{
   "edges": [
      {
         "goal": {},
         "rules": [
            {
               "name": "iFD4MY7O3g",
               "queries": []
            }
         ],
         "start": {}
      }
   ],
   "nodes": [
      {
         "class": "Dk8Bg7W9LL",
         "count": 47,
         "queries": [
            {
               "count": 60,
               "query": {},
               "statuses": []
            }
         ],
         "result": [
            {}
         ]
      }
   ]
}
```

#### Field Definitions

- `edges` *(array of Edge)* List of graph edges.
- `nodes` *(array of Node)* List of graph nodes.

**Edge**
- `start`: Class name of the start node.
- `goal`: Class name of the goal node.
- `rules` *(array of Rule)*: Set of rules followed along this edge.

**Rule**
- `name` *(string, required)*: Name is an optional descriptive name.
- `queries` *(array of QueryCount)*: Queries generated while following this rule.

**QueryCount**
- `count` *(integer)*: Number of results, omitted if the query was not executed.
- `query`: Query for correlation data.
- `statuses` *(array of StatusCount)*: Statuses found on data objects for this query.

**StatusCount**
- `status` *(string, required)*: Status for correlation data.
- `count` *(integer)*: Number of instances found, omitted if none.

**Node**
- `class` *(string, required)*: Full class name.
- `queries` *(array of QueryCount)*: Queries yielding results for this class.
- `count` *(integer)*: Number of results for this class, after de-duplication.
- `result` *(array of Object)*: Serialized result contents, may be large.

**QueryCount**
- `count` *(integer)*: Number of results, omitted if the query was not executed.
- `query`: Query for correlation data.
- `statuses` *(array of StatusCount)*: Statuses found on data objects for this query.

**StatusCount**
- `status` *(string, required)*: Status for correlation data.
- `count` *(integer)*: Number of instances found, omitted if none.

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

result not found

```json
{
   "error": "An error occurred"
}
```

### POST /graphs/neighbors {#postgraphsneighbors}

Specify a set of start objects, as queries or serialized objects, and a depth for the neighborhood search. Returns a graph of all paths with depth or less edges leading from start objects.


#### Query Parameters

- `options` *(object)* Options controlling the form of the returned graph.

### Request

```json
{
   "depth": 55,
   "start": {
      "class": {},
      "constraint": {
         "end": "2017-07-21T17:32:28.1341231Z",
         "limit": 36,
         "start": "2024-01-15T10:30:00Z"
      },
      "objects": [
         {}
      ],
      "queries": [
         "k8s:Pod:{\"namespace\":\"default\",\"name\":\"my-pod\"}"
      ]
   }
}
```

#### Field Definitions

- `depth` *(integer, required)* Maximum number of correlation steps to follow from the start. Depth 1 returns direct correlations only.

- `start` Starting point for the search.

### Responses

#### 200 Response

OK

```json
{
   "edges": [
      {
         "goal": {},
         "rules": [
            {
               "name": "iFD4MY7O3g",
               "queries": []
            }
         ],
         "start": {}
      }
   ],
   "nodes": [
      {
         "class": "Dk8Bg7W9LL",
         "count": 47,
         "queries": [
            {
               "count": 60,
               "query": {},
               "statuses": []
            }
         ],
         "result": [
            {}
         ]
      }
   ]
}
```

#### Field Definitions

- `edges` *(array of Edge)* List of graph edges.
- `nodes` *(array of Node)* List of graph nodes.

**Edge**
- `start`: Class name of the start node.
- `goal`: Class name of the goal node.
- `rules` *(array of Rule)*: Set of rules followed along this edge.

**Rule**
- `name` *(string, required)*: Name is an optional descriptive name.
- `queries` *(array of QueryCount)*: Queries generated while following this rule.

**QueryCount**
- `count` *(integer)*: Number of results, omitted if the query was not executed.
- `query`: Query for correlation data.
- `statuses` *(array of StatusCount)*: Statuses found on data objects for this query.

**StatusCount**
- `status` *(string, required)*: Status for correlation data.
- `count` *(integer)*: Number of instances found, omitted if none.

**Node**
- `class` *(string, required)*: Full class name.
- `queries` *(array of QueryCount)*: Queries yielding results for this class.
- `count` *(integer)*: Number of results for this class, after de-duplication.
- `result` *(array of Object)*: Serialized result contents, may be large.

**QueryCount**
- `count` *(integer)*: Number of results, omitted if the query was not executed.
- `query`: Query for correlation data.
- `statuses` *(array of StatusCount)*: Statuses found on data objects for this query.

**StatusCount**
- `status` *(string, required)*: Status for correlation data.
- `count` *(integer)*: Number of instances found, omitted if none.

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

result not found

```json
{
   "error": "An error occurred"
}
```

### POST /graphs/neighbours {#postgraphsneighbours}

Specify a set of start objects, as queries or serialized objects, and a depth for the neighborhood search. Returns a graph of all paths with depth or less edges leading from start objects.


#### Query Parameters

- `options` *(object)* Options controlling the form of the returned graph.

### Request

```json
{
   "depth": 55,
   "start": {
      "class": {},
      "constraint": {
         "end": "2017-07-21T17:32:28.1341231Z",
         "limit": 36,
         "start": "2024-01-15T10:30:00Z"
      },
      "objects": [
         {}
      ],
      "queries": [
         "k8s:Pod:{\"namespace\":\"default\",\"name\":\"my-pod\"}"
      ]
   }
}
```

#### Field Definitions

- `depth` *(integer, required)* Maximum number of correlation steps to follow from the start. Depth 1 returns direct correlations only.

- `start` Starting point for the search.

### Responses

#### 200 Response

OK

```json
{
   "edges": [
      {
         "goal": {},
         "rules": [
            {
               "name": "iFD4MY7O3g",
               "queries": []
            }
         ],
         "start": {}
      }
   ],
   "nodes": [
      {
         "class": "Dk8Bg7W9LL",
         "count": 47,
         "queries": [
            {
               "count": 60,
               "query": {},
               "statuses": []
            }
         ],
         "result": [
            {}
         ]
      }
   ]
}
```

#### Field Definitions

- `edges` *(array of Edge)* List of graph edges.
- `nodes` *(array of Node)* List of graph nodes.

**Edge**
- `start`: Class name of the start node.
- `goal`: Class name of the goal node.
- `rules` *(array of Rule)*: Set of rules followed along this edge.

**Rule**
- `name` *(string, required)*: Name is an optional descriptive name.
- `queries` *(array of QueryCount)*: Queries generated while following this rule.

**QueryCount**
- `count` *(integer)*: Number of results, omitted if the query was not executed.
- `query`: Query for correlation data.
- `statuses` *(array of StatusCount)*: Statuses found on data objects for this query.

**StatusCount**
- `status` *(string, required)*: Status for correlation data.
- `count` *(integer)*: Number of instances found, omitted if none.

**Node**
- `class` *(string, required)*: Full class name.
- `queries` *(array of QueryCount)*: Queries yielding results for this class.
- `count` *(integer)*: Number of results for this class, after de-duplication.
- `result` *(array of Object)*: Serialized result contents, may be large.

**QueryCount**
- `count` *(integer)*: Number of results, omitted if the query was not executed.
- `query`: Query for correlation data.
- `statuses` *(array of StatusCount)*: Statuses found on data objects for this query.

**StatusCount**
- `status` *(string, required)*: Status for correlation data.
- `count` *(integer)*: Number of instances found, omitted if none.

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

result not found

```json
{
   "error": "An error occurred"
}
```

### POST /lists/goals {#postlistsgoals}

Specify a set of start objects, as queries or serialized objects, and a goal class. Returns a list of all objects of the goal class that can be reached from a start object.


### Request

```json
{
   "goals": [
      "k8s:Pod",
      "metric:metric"
   ],
   "start": {
      "class": {},
      "constraint": {
         "end": "2017-07-21T17:32:28.1341231Z",
         "limit": 12,
         "start": "2024-01-15T10:30:00Z"
      },
      "objects": [
         {}
      ],
      "queries": [
         "k8s:Pod:{\"namespace\":\"default\",\"name\":\"my-pod\"}"
      ]
   }
}
```

#### Field Definitions

- `goals` *(array of Class, required)* Goal classes in DOMAIN:CLASS format, e.g. log:application, alert:alert

- `start` Starting point for the search.

### Responses

#### 200 Response

OK

```json
[
   {
      "class": "GNO6q1Xh3S",
      "count": 81,
      "queries": [
         {
            "count": 76,
            "query": {},
            "statuses": []
         }
      ],
      "result": [
         {}
      ]
   }
]
```

#### Field Definitions

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

result not found

```json
{
   "error": "An error occurred"
}
```

## query

### GET /domains {#getdomains}

Returns a list of Korrel8r domains and the stores configured for each domain.


### Responses

#### 200 Response

OK

```json
[
   {
      "description": "yAVmNkB33i",
      "name": "5zQu9MxNmG",
      "stores": [
         {}
      ]
   }
]
```

#### Field Definitions

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

result not found

```json
{
   "error": "An error occurred"
}
```

### GET /domain/{domain}/classes {#getdomaindomainclasses}

Returns a list of class names for the specified domain.


#### Path Parameters

- `domain` *(string, required)* Name of the domain to list classes for

### Responses

#### 200 Response

OK

```json
[
   "Pod",
   "Service",
   "Deployment"
]
```

#### Field Definitions

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

domain not found

```json
{
   "error": "An error occurred"
}
```

### GET /objects {#getobjects}

Execute a single Korrel8r 'query' and return the list of serialized objects found. Does not perform any correlation actions.


#### Query Parameters

- `query` *(string, required)* Query string.

- `constraint` *(object)* Constrains the objects that will be included in results.

### Responses

#### 200 Response

OK

```json
[
   {}
]
```

#### Field Definitions

#### 400 Response

invalid parameters

```json
{
   "error": "An error occurred"
}
```

#### 404 Response

result not found

```json
{
   "error": "An error occurred"
}
```

