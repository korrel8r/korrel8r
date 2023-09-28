


# REST API for korrel8r
  

## Informations

### Version

v1alpha1

### Contact

  https://github.com/korrel8r/korrel8r

## Content negotiation

### URI Schemes
  * http

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  configuration

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /api/v1alpha1/domains | [get domains](#get-domains) | List all configured domains and stores. |
| GET | /api/v1alpha1/domains/{domain}/classes | [get domains domain classes](#get-domains-domain-classes) | Get class names and descriptions for the domain. |
  


###  search

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| POST | /api/v1alpha1/graphs/goals | [post graphs goals](#post-graphs-goals) | Create a correlation graph from start objects to goal queries. |
| POST | /api/v1alpha1/graphs/neighbours | [post graphs neighbours](#post-graphs-neighbours) | Create a correlation graph of neighbours of a start object to a given depth. |
| POST | /api/v1alpha1/lists/goals | [post lists goals](#post-lists-goals) | Generate a list of goal nodes related to a starting point. |
  


## Paths

### <span id="get-domains"></span> List all configured domains and stores. (*GetDomains*)

```
GET /api/v1alpha1/domains
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-domains-200) | OK | OK |  | [schema](#get-domains-200-schema) |

#### Responses


##### <span id="get-domains-200"></span> 200 - OK
Status: OK

###### <span id="get-domains-200-schema"></span> Schema
   
  

[][APIDomain](#api-domain)

### <span id="get-domains-domain-classes"></span> Get class names and descriptions for the domain. (*GetDomainsDomainClasses*)

```
GET /api/v1alpha1/domains/{domain}/classes
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| domain | `path` | string | `string` |  | ✓ |  | Domain to get classes from. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-domains-domain-classes-200) | OK | OK |  | [schema](#get-domains-domain-classes-200-schema) |

#### Responses


##### <span id="get-domains-domain-classes-200"></span> 200 - OK
Status: OK

###### <span id="get-domains-domain-classes-200-schema"></span> Schema
   
  

[APIClasses](#api-classes)

### <span id="post-graphs-goals"></span> Create a correlation graph from start objects to goal queries. (*PostGraphsGoals*)

```
POST /api/v1alpha1/graphs/goals
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| withRules | `query` | boolean | `bool` |  |  |  | include rules in graph edges |
| start | `body` | [APIGoalsRequest](#api-goals-request) | `models.APIGoalsRequest` | | ✓ | | search from start to goal classes |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-graphs-goals-200) | OK | OK |  | [schema](#post-graphs-goals-200-schema) |

#### Responses


##### <span id="post-graphs-goals-200"></span> 200 - OK
Status: OK

###### <span id="post-graphs-goals-200-schema"></span> Schema
   
  

[APIGraph](#api-graph)

### <span id="post-graphs-neighbours"></span> Create a correlation graph of neighbours of a start object to a given depth. (*PostGraphsNeighbours*)

```
POST /api/v1alpha1/graphs/neighbours
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| withRules | `query` | boolean | `bool` |  |  |  | include rules in graph edges |
| start | `body` | [APINeighboursRequest](#api-neighbours-request) | `models.APINeighboursRequest` | | ✓ | | search from neighbours |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-graphs-neighbours-200) | OK | OK |  | [schema](#post-graphs-neighbours-200-schema) |

#### Responses


##### <span id="post-graphs-neighbours-200"></span> 200 - OK
Status: OK

###### <span id="post-graphs-neighbours-200-schema"></span> Schema
   
  

[APIGraph](#api-graph)

### <span id="post-lists-goals"></span> Generate a list of goal nodes related to a starting point. (*PostListsGoals*)

```
POST /api/v1alpha1/lists/goals
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| start | `body` | [APIGoalsRequest](#api-goals-request) | `models.APIGoalsRequest` | | ✓ | | search from start to goal classes |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-lists-goals-200) | OK | OK |  | [schema](#post-lists-goals-200-schema) |

#### Responses


##### <span id="post-lists-goals-200"></span> 200 - OK
Status: OK

###### <span id="post-lists-goals-200-schema"></span> Schema
   
  

[][APINode](#api-node)

## Models

### <span id="api-classes"></span> api.Classes


> Classes maps class names to a short description.
  



[APIClasses](#api-classes)

### <span id="api-domain"></span> api.Domain


> Domain configuration information.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| errors | []string| `[]string` |  | |  |  |
| name | string| `string` |  | |  |  |
| stores | [][Korrel8rStoreConfig](#korrel8r-store-config)| `[]Korrel8rStoreConfig` |  | |  |  |



### <span id="api-edge"></span> api.Edge


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| goal | string| `string` |  | | Goal is the class name of the goal node. | `class.domain` |
| rules | [][APIRule](#api-rule)| `[]*APIRule` |  | | Rules is the set of rules followed along this edge (optional). |  |
| start | string| `string` |  | | Start is the class name of the start node. |  |



### <span id="api-goals-request"></span> api.GoalsRequest


> Starting point for a goals search.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| goals | []string| `[]string` |  | | Goal classes for correlation. | `["class.domain"]` |
| start | [APIGoalsRequest](#api-goals-request)| `APIGoalsRequest` |  | | Start of correlation search. |  |



### <span id="api-graph"></span> api.Graph


> Graph resulting from a correlation search.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| edges | [][APIEdge](#api-edge)| `[]*APIEdge` |  | |  |  |
| nodes | [][APINode](#api-node)| `[]*APINode` |  | |  |  |



### <span id="api-neighbours-request"></span> api.NeighboursRequest


> Starting point for a neighbours search.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| depth | integer| `int64` |  | | Max depth of neighbours graph. |  |
| start | [APINeighboursRequest](#api-neighbours-request)| `APINeighboursRequest` |  | | Start of correlation search. |  |



### <span id="api-node"></span> api.Node


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| class | string| `string` |  | | Class is the full name of the class in "CLASS.DOMAIN" form. | `class.domain` |
| count | integer| `int64` |  | | Count of results found for this class, after de-duplication. |  |
| queries | [APINode](#api-node)| `APINode` |  | | Queries yielding results for this class. | `{"querystring":10}` |



### <span id="api-queries"></span> api.Queries


> A set of query strings with counts of results found by the query. Value of -1 means the query was not run so result count is unknown.
  



[APIQueries](#api-queries)

### <span id="api-rule"></span> api.Rule


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name is an optional descriptive name. |  |
| queries | [APIRule](#api-rule)| `APIRule` |  | | Queries generated while following this rule. | `{"querystring":10}` |



### <span id="api-start"></span> api.Start


> Starting point for correlation.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| class | string| `string` |  | | Class of starting objects | `class.domain` |
| objects | [interface{}](#interface)| `interface{}` |  | | Objects in JSON form |  |
| query | []string| `[]string` |  | | Queries for starting objects |  |



### <span id="korrel8r-store-config"></span> korrel8r.StoreConfig


  

[Korrel8rStoreConfig](#korrel8r-store-config)
