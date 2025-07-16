// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rest implements a REST API for korrel8r.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

var log = logging.Log()

type API struct {
	Engine  *engine.Engine
	Configs config.Configs
	Router  *gin.Engine
}

var _ ServerInterface = &API{}

// New API instance, registers handlers with a gin Engine.
func New(e *engine.Engine, c config.Configs, r *gin.Engine) (*API, error) {
	a := &API{Engine: e, Configs: c, Router: r}
	r.Use(a.logger)
	r.Use(a.context)
	basePath, _ := Spec().Servers.BasePath()
	RegisterHandlersWithOptions(r, a, GinServerOptions{BaseURL: basePath})
	return a, nil
}

func (a *API) ListDomains(c *gin.Context) {
	var domains []Domain
	for _, d := range a.Engine.Domains() {
		var stores []Store
		for _, sc := range a.Engine.StoreConfigsFor(d) {
			stores = append(stores, (Store)(sc))
		}
		domains = append(domains, Domain{
			Name:   d.Name(),
			Stores: stores,
		})
	}
	c.JSON(http.StatusOK, domains)
}

func (a *API) GraphGoals(c *gin.Context, params GraphGoalsParams) {
	g, _ := a.goals(c)
	if c.IsAborted() {
		return
	}
	gr := Graph{Nodes: nodes(g), Edges: edges(g, ptr.Deref(params.Rules))}
	okResponse(c, gr)
}

func (a *API) ListGoals(c *gin.Context) {
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

func (a *API) GraphNeighbours(c *gin.Context, params GraphNeighboursParams) {
	r := Neighbours{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
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
	gr := Graph{Nodes: nodes(g), Edges: edges(g, ptr.Deref(params.Rules))}
	if !interrupted(c) {
		check(c, http.StatusBadRequest, err)
	}
	okResponse(c, gr)
}

func (a *API) Query(c *gin.Context, params QueryParams) {
	query, err := a.Engine.Query(params.Query)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	constraint := (*korrel8r.Constraint)(nil) // TODO can't pass constraints
	result := result.New(query.Class())
	if !check(c, http.StatusNotFound, a.Engine.Get(c.Request.Context(), query, constraint, result)) {
		return
	}
	log.V(3).Info("Response OK", "objects", len(result.List()))
	body := []any(result.List())
	if body == nil {
		body = []any{} // Return [] on empty, not null.
	}
	c.JSON(http.StatusOK, body)
}

func (a *API) SetConfig(c *gin.Context, params SetConfigParams) {
	if params.Verbose != nil {
		log.V(1).Info("Config set verbose", "level", *params.Verbose)
		logging.SetVerbose(*params.Verbose)
	}
	c.JSON(http.StatusOK, params)
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

func (a *API) queries(c *gin.Context, queryStrings []string) []korrel8r.Query {
	var queries []korrel8r.Query
	for _, q := range queryStrings {
		query, err := a.Engine.Query(q)
		if !check(c, http.StatusBadRequest, err, "query parameter") {
			return nil
		}
		queries = append(queries, query)
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
		if class == nil {
			return traverse.Start{}, nil
		}
	} else {
		class = a.class(c, start.Class)
	}
	if class == nil {
		return traverse.Start{}, nil
	}
	objects := a.objects(c, class, start.Objects)
	return traverse.Start{Class: class, Objects: objects, Queries: queries}, constraint(start.Constraint)
}

func constraint(c *Constraint) *korrel8r.Constraint {
	if c == nil {
		return nil
	}
	return &korrel8r.Constraint{
		Start:   c.Start,
		End:     c.End,
		Limit:   c.Limit,
		Timeout: c.Timeout,
	}
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if err != nil && !c.IsAborted() {
		if len(format) > 0 {
			err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
		}
		ginErr := c.Error(err)
		if !traverse.IsPartialError(err) { // Don't abort on a partial error.
			c.AbortWithStatusJSON(code, ginErr.JSON())
			log.Error(err, "Request failed", "url", c.Request.URL, "code", code, "error", err)
		}
	}
	return err == nil && !c.IsAborted()
}

func (a *API) class(c *gin.Context, className string) korrel8r.Class {
	class, err := a.Engine.Class(className)
	check(c, http.StatusBadRequest, err)
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
	if c.IsAborted() {
		return
	}
	status := http.StatusOK
	if interrupted(c) {
		status = http.StatusPartialContent
	}
	c.JSON(status, body)
}
