// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package api implements a REST API for korrel8r.
//
// See generated doc at https://github.com/korrel8r/korrel8r/blob/main/pkg/api/docs/swagger.md
//
// Note: Comments starting with "@" are used to generate a swagger spec via 'go generate'.
// Static swagger doc is available at ./docs/swagger.md.
// dynamic HTML doc is available from korrel8r at the "/api" endpoint.
//
//	@title			REST API for korrel8r
//	@version		v1alpha1
//	@contact.url	https://github.com/korrel8r/korrel8r
//	@basePath		/api/v1alpha1
//	@accept			json
//	@produce		json
//
//go:generate swag init -g api.go
//go:generate swag fmt -d ./
//go:generate swagger generate markdown -f docs/swagger.json --output ../../doc/rest-api.md
//go:generate cp docs/swagger.json ../../doc
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/pkg/api/docs"
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
	v.GET("/stores", a.GetStores)
	v.GET("/stores/:domain", a.GetStoresDomain)
	v.POST("/graph/goals", a.GraphGoals)
	v.POST("/list/goals", a.ListGoals)
	v.POST("/graph/neighbours", a.GraphNeighbours)
	return a, nil
}

// Close cleans any persistent resources.
func (a *API) Close() {}

// GetDomains handler
//
//	@router		/domains [get]
//	@summary	List all korrel8r domain names.
//	@tags		configuration
//	@success	200	{array}	string
func (a *API) GetDomains(c *gin.Context) {
	var domains []string
	for _, d := range a.Engine.Domains() {
		domains = append(domains, d.String())
	}
	c.JSON(http.StatusOK, domains)
}

// GetStores handler
//
//	@router		/stores [get]
//	@summary	List of all store configurations objects.
//	@tags		configuration
//	@success	200	{array}	StoreConfig
func (a *API) GetStores(c *gin.Context) {
	var stores []StoreConfig
	for _, d := range a.Engine.Domains() {
		for _, c := range a.Engine.StoreConfigsFor(d) {
			stores = append(stores, StoreConfig(c))
		}
	}
	c.JSON(http.StatusOK, stores)
}

// GetStoresDomain handler
//
//	@router		/stores/{domain} [get]
//	@summary	List of all store configurations objects for a specific domain.
//	@tags		configuration
//	@param		domain	path	string	true	"domain	name"
//	@success	200		{array}	StoreConfig
func (a *API) GetStoresDomain(c *gin.Context) {
	name := strings.TrimPrefix(c.Params.ByName("domain"), "/")
	d, err := a.Engine.DomainErr(name)
	if check(c, http.StatusNotFound, err) {
		c.JSON(http.StatusOK, a.Engine.StoreConfigsFor(d))
	}
}

// GraphGoals handler.
//
//	@router		/graph/goals [post]
//	@summary	Create a correlation graph from start objects to goal queries.
//	@param		withRules	query	bool			false	"include rules in graph edges"
//	@param		start		body	GoalsRequest	true	"search from start to goal classes"
//	@tags		search
//	@success	200	{object}	Graph
func (a *API) GraphGoals(c *gin.Context) { // FIXME consistent naming of endpoints. GraphGoals
	opts := &Options{}
	if check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		if g, _ := a.goalsRequest(c); g != nil {
			c.JSON(http.StatusOK, newGraph(g, opts))
		}
	}
}

// ListGoals handler.
//
//	@router		/list/goals [post]
//	@summary	Generate a list of goal nodes related to a starting point.
//	@param		start	body	GoalsRequest	true	"search from start to goal classes"
//	@tags		search
//	@success	200	{array}	Node
func (a *API) ListGoals(c *gin.Context) {
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

// GraphNeighbours handler
//
//	@router		/graph/neighbours [post]
//	@summary	Create a correlation graph of neighbours of a start object to a given depth.
//	@param		withRules	query	bool				false	"include rules in graph edges"
//	@param		start		body	NeighboursRequest	true	"search from neighbours"
//	@tags		search
//	@success	200	{object}	Graph
func (a *API) GraphNeighbours(c *gin.Context) {
	r, opts := NeighboursRequest{}, Options{}
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
	c.JSON(http.StatusOK, newGraph(g, &opts))
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

func (a *API) queries(c *gin.Context, class korrel8r.Class, queryStrings []string) (queries []korrel8r.Query) {
	for _, q := range queryStrings {
		query, err := class.Domain().Query(q)
		if check(c, http.StatusBadRequest, err, "query parameter") {
			queries = append(queries, query)
		}
	}
	return queries
}
func (a *API) objects(c *gin.Context, class korrel8r.Class, raw []json.RawMessage) (objects []korrel8r.Object) {
	for _, r := range raw {
		obj := class.New()
		if !check(c, http.StatusBadRequest, json.Unmarshal([]byte(r), &obj), "decoding object of class %v", classname(class)) {
			return nil
		}
		objects = append(objects, obj)
	}
	return objects
}

// start validates and extracts data from the Start part of a request.
func (a *API) start(c *gin.Context, start *Start) (korrel8r.Class, []korrel8r.Object, []korrel8r.Query) {
	class := a.class(c, start.Class)
	objects := a.objects(c, class, start.Objects)
	queries := a.queries(c, class, start.Queries)
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
		if check(c, http.StatusBadRequest, a.Engine.Get(c.Request.Context(), start, query, cr),
			"query failed: %q", query.String) {
			n.QueryCounts.Put(query, cr.Count)
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

// FIXME
// Link from project page.
// FIXME
// Review all.
