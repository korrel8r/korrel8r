// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rest implements a REST API for korrel8r.
//
// Dynamic HTML doc is available from korrel8r at the "/api" endpoint.
//
// Note: Comments starting with "@" are used to generate a swagger spec.
//
//	@title			REST API
//	@description	REST API for the Korrel8r correlation engine.
//	@version		v1alpha1
//	@license.name	Apache 2.0
//	@license.url	https://github.com/korrel8r/korrel8r/blob/main/LICENSE
//	@contact.name	Project Korrel8r
//	@contact.url	https://github.com/korrel8r/korrel8r
//	@host			localhost:8080
//	@basePath		/api/v1alpha1
//	@schemes		https http
//	@accept			json
//	@produce		json
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
	"github.com/korrel8r/korrel8r/pkg/rest/docs"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/unique"
	swaggofiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

var log = logging.Log()

// BasePath is the versioned base path for the current version of the REST API.
var BasePath = docs.SwaggerInfo.BasePath

type API struct {
	Engine  *engine.Engine
	Configs config.Configs
	Router  *gin.Engine
}

// New API instance, registers  handlers with a gin Engine.
func New(e *engine.Engine, c config.Configs, r *gin.Engine) (*API, error) {
	a := &API{Engine: e, Configs: c, Router: r}
	r.Use(a.logger)
	r.Use(a.context)
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html") })
	r.GET("/api", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html") })
	r.GET("/swagger/*any", a.handleSwagger)
	v := r.Group(docs.SwaggerInfo.BasePath)
	v.GET("/domains", a.Domains)
	v.GET("/objects", a.GetObjects)
	v.POST("/graphs/goals", a.GraphsGoals)
	v.POST("/graphs/neighbours", a.GraphsNeighbours)
	v.POST("/lists/goals", a.ListsGoals)
	v.PUT("/config", a.PutConfig)
	return a, nil
}

// Close cleans any persistent resources.
func (a *API) Close() {}

func (a *API) handleSwagger(c *gin.Context) {
	// Set the SwaggerInfo Host to be consistent with the incoming request URL so the test UI will work.
	// Note this may not work properly if there are concurrent requests with different URLs.
	swaggerInfoLock.Lock()
	docs.SwaggerInfo.Host = c.Request.URL.Host
	defer swaggerInfoLock.Unlock()
	ginswagger.WrapHandler(swaggofiles.Handler)(c)
}

var swaggerInfoLock sync.Mutex

// Domains handler
//
//	@router		/domains [get]
//	@summary	Get name, configuration and status for each domain.
//	@success	200		{array}		Domain
//	@failure	default	{object}	any
func (a *API) Domains(c *gin.Context) {
	var domains []Domain
	for _, d := range a.Engine.Domains() {
		domains = append(domains, Domain{
			Name:   d.Name(),
			Stores: a.Engine.StoreConfigsFor(d),
		})
	}
	c.JSON(http.StatusOK, domains)
}

// GraphsGoals handler.
//
//	@router		/graphs/goals [post]
//	@summary	Create a correlation graph from start objects to goal queries.
//	@param		rules	query		bool	false	"include rules in graph edges"
//	@param		request	body		Goals	true	"search from start to goal classes"
//	@success	200		{object}	Graph
//	@success	206		{object}	Graph "interrupted, partial result"
//	@failure	default	{object}	any
func (a *API) GraphsGoals(c *gin.Context) {
	opts := &Options{}
	if !check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		return
	}
	g, _ := a.goals(c)
	if c.IsAborted() {
		return
	}
	gr := Graph{Nodes: nodes(g), Edges: edges(g, *opts)}
	okResponse(c, gr)
}

// ListsGoals handler.
//
//	@router		/lists/goals [post]
//	@summary	Create a list of goal nodes related to a starting point.
//	@param		request	body		Goals	true	"search from start to goal classes"
//	@success	200		{array}		Node
//	@failure	default	{object}	any
func (a *API) ListsGoals(c *gin.Context) {
	nodes := []Node{} // return [] not null for empty
	g, goals := a.goals(c)
	if c.IsAborted() {
		return
	}
	set := unique.NewSet(goals...)
	g.EachNode(func(n *graph.Node) {
		if set.Has(n.Class) {
			nodes = append(nodes, node(n))
		}
	})
	okResponse(c, nodes)
}

// GraphsNeighbours handler
//
//	@router		/graphs/neighbours [post]
//	@summary	Create a neighbourhood graph around a start object to a given depth.
//	@param		rules	query		bool		false	"include rules in graph edges"
//	@param		request	body		Neighbours	true	"search from neighbours"
//	@success	200		{object}	Graph
//	@success	206		{object}	Graph "interrupted, partial result"
//	@failure	default	{object}	any
func (a *API) GraphsNeighbours(c *gin.Context) {
	r, opts := Neighbours{}, Options{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) || !check(c, http.StatusBadRequest, c.BindUri(&opts)) {
		return
	}
	start, constraint := a.start(c, &r.Start)
	depth := r.Depth
	if c.IsAborted() {
		return
	}
	ctx, cancel := korrel8r.WithConstraint(c.Request.Context(), constraint.Default())
	defer cancel()
	g, err := traverse.New(a.Engine, a.Engine.Graph()).Neighbours(ctx, start, depth)
	gr := Graph{Nodes: nodes(g), Edges: edges(g, opts)}
	if !interrupted(c) {
		check(c, http.StatusBadRequest, err)
	}
	if !c.IsAborted() {
		okResponse(c, gr)
	}
}

// GetObjects handler
//
//	@router		/objects [get]
//	@summary	Execute a query, returns a list of JSON objects.
//	@param		query	query		string	true	"query string"
//	@success	200		{array}		any
//	@failure	default	{object}	any
func (a *API) GetObjects(c *gin.Context) {
	opts := &Objects{}
	if !check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		return
	}
	query, err := a.Engine.Query(opts.Query)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	result := result.New(query.Class())
	if !check(c, http.StatusNotFound, a.Engine.Get(c.Request.Context(), query, (*korrel8r.Constraint)(opts.Constraint), result)) {
		return
	}
	log.V(3).Info("REST: response OK", "objects", len(result.List()))
	body := []any(result.List())
	if body == nil {
		body = []any{} // Return [] on empty, not null.
	}
	c.JSON(http.StatusOK, body)
}

func (a *API) goals(c *gin.Context) (g *graph.Graph, goals []korrel8r.Class) {
	r := Goals{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return nil, nil
	}
	start, constraint := a.start(c, &r.Start)
	goals = a.classes(c, r.Goals)
	if c.IsAborted() {
		return nil, nil
	}
	g = a.Engine.Graph().ShortestPaths(start.Class, goals...)
	var err error
	ctx, cancel := korrel8r.WithConstraint(c.Request.Context(), constraint.Default())
	defer cancel()
	g, err = traverse.New(a.Engine, g).Goals(ctx, start, goals)
	if !interrupted(c) && !traverse.IsPartialError(err) {
		check(c, http.StatusNotFound, err)
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
		o, err := class.Unmarshal([]byte(r))
		if !check(c, http.StatusBadRequest, err, "decoding object of class %v", class.String()) {
			return nil
		}
		objects = append(objects, o)
	}
	return objects
}

// start validates and extracts data from the Start part of a request.
func (a *API) start(c *gin.Context, start *Start) (traverse.Start, *korrel8r.Constraint) {
	queries := a.queries(c, start.Queries)
	var class korrel8r.Class
	if start.Class == "" && len(queries) > 0 {
		class = queries[0].Class()
	} else {
		class = a.class(c, start.Class)
	}
	if class == nil {
		return traverse.Start{}, nil
	}
	objects := a.objects(c, class, start.Objects)
	return traverse.Start{Class: class, Objects: objects, Queries: queries}, start.Constraint
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if err != nil && !c.IsAborted() {
		if len(format) > 0 {
			err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
		}
		ginErr := c.Error(err)
		if !traverse.IsPartialError(err) { // Don't abort on a partial error.
			c.AbortWithStatusJSON(code, ginErr.JSON())
			log.Error(err, "REST: request failed", "url", c.Request.URL, "code", code, "error", err)
		}
	}
	return err == nil && !c.IsAborted()
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

// logger is a Gin handler to log requests.
func (a *API) logger(c *gin.Context) {
	start := time.Now()
	c.Next()
	log := log.WithValues(
		"method", c.Request.Method,
		"url", c.Request.URL,
		"from", c.Request.RemoteAddr,
		"status", c.Writer.Status(),
		"latency", time.Since(start))
	if interrupted(c) {
		log = log.WithValues("interrupted", c.Request.Context().Err())
	}
	if c.IsAborted() {
		log = log.WithValues("errors", c.Errors.Errors())
	}
	if c.IsAborted() || c.Writer.Status() > 500 {
		log.V(2).Info("REST: request failed")
	} else {
		log.V(3).Info("REST: request OK")
	}
}

// context sets up authorization and deadline context for outgoing requests.
func (a *API) context(c *gin.Context) {
	ctx := auth.Context(c.Request) // add authentication

	timeout := korrel8r.DefaultTimeout
	if len(a.Configs) > 0 {
		tuning := a.Configs[0].Tuning
		if tuning != nil && tuning.RequestTimeout.Duration > 0 {
			timeout = tuning.RequestTimeout.Duration
		}
	}
	deadline := time.Now().Add(timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	c.Request = c.Request.WithContext(ctx)
	defer cancel()

	c.Next()
}

func interrupted(c *gin.Context) bool {
	return c.Request.Context().Err() == context.DeadlineExceeded ||
		c.Errors.Last() != nil && traverse.IsPartialError(c.Errors.Last())
}

func okResponse(c *gin.Context, body any) {
	status := http.StatusOK
	if interrupted(c) {
		status = http.StatusPartialContent
	}
	c.JSON(status, body)
}
