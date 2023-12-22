// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package api implements a REST API for korrel8r.
//
// Note: Comments starting with "@" are used to generate a swagger spec via 'go generate'.
// Static swagger doc is available at ./docs/swagger.md.
// dynamic HTML doc is available from korrel8r at the "/api" endpoint.
//
//	@title			REST API
//	@description	REST API for the Korrel8r correlation engine.
//	@version		v1alpha1
//	@license.name	Apache 2.0
//	@license.url	https://github.com/korrel8r/korrel8r/blob/main/LICENSE
//	@contact.name	Project Korrel8r
//	@contact.url	https://github.com/korrel8r/korrel8r
//	@basePath		/api/v1alpha1
//	@schemes		http https
//	@accept			json
//	@produce		json
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	docs "github.com/korrel8r/korrel8r/pkg/api/zz_docs"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	swaggoFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// BasePath is the versioned base path for the current version of the REST API.
var BasePath = docs.SwaggerInfo.BasePath

type API struct {
	Engine *engine.Engine
}

// New API instance, registers  handlers with a gin Engine.
func New(e *engine.Engine, r *gin.Engine) (*API, error) {
	a := &API{Engine: e}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggoFiles.Handler))
	r.GET("/api", func(c *gin.Context) { c.Redirect(http.StatusPermanentRedirect, "/swagger/index.html") })
	v := r.Group(docs.SwaggerInfo.BasePath)
	v.GET("/domains", a.GetDomains)
	v.GET("/domains/:domain/classes", a.GetDomainClasses)
	v.POST("/graphs/goals", a.GraphsGoals)
	v.POST("/lists/goals", a.ListsGoals)
	v.POST("/graphs/neighbours", a.GraphsNeighbours)
	v.GET("/objects", a.GetObjects)
	return a, nil
}

// Close cleans any persistent resources.
func (a *API) Close() {}

// GetDomains handler
//
//	@router		/domains [get]
//	@summary	List all configured domains and stores.
//	@tags		configuration
//	@success	200	{array}	Domain
func (a *API) GetDomains(c *gin.Context) {
	var domains []Domain
	for _, d := range a.Engine.Domains() {
		domains = append(domains, Domain{
			Name:   d.Name(),
			Stores: a.Engine.StoreConfigsFor(d),
		})
	}
	c.JSON(http.StatusOK, domains)
}

// GetDomainClasses handler
//
//	@router		/domains/{domain}/classes [get]
//	@summary	Get class names and descriptions for the domain.
//	@param		domain	path	string	true	"Domain to get classes from."
//	@tags		configuration
//	@success	200	{object}	Classes
func (a *API) GetDomainClasses(c *gin.Context) {
	d, err := a.Engine.DomainErr(c.Params.ByName("domain"))
	check(c, http.StatusNotFound, err)
	classes := Classes{}
	for _, c := range d.Classes() {
		classes[c.Name()] = c.Description()
	}
	c.JSON(http.StatusOK, classes)
}

// GraphsGoals handler.
//
//	@router		/graphs/goals [post]
//	@summary	Create a correlation graph from start objects to goal queries.
//	@param		withRules	query	bool			false	"include rules in graph edges"
//	@param		start		body	GoalsRequest	true	"search from start to goal classes"
//	@tags		search
//	@success	200	{object}	Graph
func (a *API) GraphsGoals(c *gin.Context) {
	opts := &GraphOptions{}
	if check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		if g, _ := a.goalsRequest(c); g != nil {
			c.JSON(http.StatusOK, Graph{Nodes: nodes(g), Edges: edges(g, opts)})
		}
	}
}

// ListsGoals handler.
//
//	@router		/lists/goals [post]
//	@summary	Generate a list of goal nodes related to a starting point.
//	@param		start	body	GoalsRequest	true	"search from start to goal classes"
//	@tags		search
//	@success	200	{array}	Node
func (a *API) ListsGoals(c *gin.Context) {
	nodes := []Node{} // return [] not null for empty
	g, goals := a.goalsRequest(c)
	if g == nil {
		return
	}
	set := unique.NewSet(goals...)
	g.EachNode(func(n *graph.Node) {
		if set.Has(n.Class) {
			nodes = append(nodes, node(n))
		}
	})
	c.JSON(http.StatusOK, nodes)
}

// GraphsNeighbours handler
//
//	@router		/graphs/neighbours [post]
//	@summary	Create a correlation graph of neighbours of a start object to a given depth.
//	@param		withRules	query	bool				false	"include rules in graph edges"
//	@param		start		body	NeighboursRequest	true	"search from neighbours"
//	@tags		search
//	@success	200	{object}	Graph
func (a *API) GraphsNeighbours(c *gin.Context) {
	r, opts := NeighboursRequest{}, GraphOptions{}
	check(c, http.StatusBadRequest, c.BindJSON(&r))
	check(c, http.StatusBadRequest, c.BindUri(&opts))
	if c.Errors != nil {
		return
	}
	start, objects, queries := a.start(c, &r.Start)
	depth := r.Depth
	if c.Errors != nil {
		return
	}
	g := a.Engine.Graph()
	if !a.setupStart(c, g, start, objects, queries) {
		return
	}
	follower := a.Engine.Follower(c.Request.Context())
	g = g.Neighbours(start, depth, follower.Traverse)
	c.JSON(http.StatusOK, Graph{Nodes: nodes(g), Edges: edges(g, &opts)})
}

// GetObjects handler
//
//	@router		/objects [get]
//	@summary	Execute a query, returns a list of JSON objects.
//	@param		query	query	string	true	"query string"
//	@tags		search
//	@success	200	{array}	any
func (a *API) GetObjects(c *gin.Context) {
	opts := &QueryOptions{}
	if !check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		return
	}
	query, err := a.Engine.Query(opts.Query)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	result := korrel8r.NewResult(query.Class())
	if !check(c, http.StatusInternalServerError, a.Engine.Get(c.Request.Context(), query, result)) {
		return
	}
	c.JSON(http.StatusOK, result.List())
}

func (a *API) goalsRequest(c *gin.Context) (g *graph.Graph, goals []korrel8r.Class) {
	r := GoalsRequest{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return nil, nil
	}
	start, objects, queries := a.start(c, &r.Start)
	goals = a.classes(c, r.Goals)
	if c.Errors != nil {
		return nil, nil
	}
	g = a.Engine.Graph().AllPaths(start, goals...)
	if !a.setupStart(c, g, start, objects, queries) {
		return nil, nil
	}

	follower := a.Engine.Follower(c.Request.Context())
	if !check(c, http.StatusInternalServerError, g.Traverse(follower.Traverse)) {
		return nil, nil
	}
	return g, goals
}

func (a *API) queries(c *gin.Context, queryStrings []string) (queries []korrel8r.Query) {
	for _, q := range queryStrings {
		query, err := a.Engine.Query(q)
		if check(c, http.StatusBadRequest, err, "query parameter") {
			queries = append(queries, query)
		}
	}
	return queries
}

func (a *API) objects(c *gin.Context, class korrel8r.Class, raw []json.RawMessage) (objects []korrel8r.Object) {
	for _, r := range raw {
		obj := class.New()
		if !check(c, http.StatusBadRequest, json.Unmarshal([]byte(r), &obj), "decoding object of class %v", korrel8r.ClassName(class)) {
			return nil
		}
		objects = append(objects, obj)
	}
	return objects
}

// start validates and extracts data from the Start part of a request.
func (a *API) start(c *gin.Context, start *Start) (korrel8r.Class, []korrel8r.Object, []korrel8r.Query) {
	class := a.class(c, start.Class)
	if c.Errors != nil {
		return nil, nil, nil
	}
	objects := a.objects(c, class, start.Objects)
	queries := a.queries(c, start.Queries)
	return class, objects, queries
}

// setupStart sets up the start node of the graph
func (a *API) setupStart(c *gin.Context, g *graph.Graph, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query) (ok bool) {
	n := g.NodeFor(start)
	result := n.Result
	result.Append(objects...)
	for _, query := range queries {
		cr := korrel8r.NewCountResult(result)
		// TODO should we tolerate get failures and report in the response?
		if check(c, http.StatusBadRequest, a.Engine.Get(c.Request.Context(), query, cr),
			"query failed: %q", query.String()) {
			n.Queries[query.String()] = cr.Count
		}
	}
	return c.Errors == nil
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
		}
		c.AbortWithStatusJSON(code, c.Error(err).JSON())
	}
	return err == nil
}

func (a *API) class(c *gin.Context, className string) korrel8r.Class {
	class, err := a.Engine.Class(className)
	check(c, http.StatusNotFound, err)
	return class
}

func (a *API) classes(c *gin.Context, apiClasses []string) (classes []korrel8r.Class) {
	for _, apiClass := range apiClasses {
		if class := a.class(c, apiClass); class != nil {
			classes = append(classes, class)
		}
	}
	return classes
}
