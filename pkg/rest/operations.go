// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rest implements a REST API for korrel8r.
package rest

import (
	"context"
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
	Engine   *engine.Engine
	Configs  config.Configs
	Router   *gin.Engine
	BasePath string
}

var _ ServerInterface = &API{}

// New API instance, registers handlers with a gin Engine.
func New(e *engine.Engine, c config.Configs, r *gin.Engine) (*API, error) {
	a := &API{Engine: e, Configs: c, Router: r}
	a.BasePath = BasePath
	r.Use(a.logger, a.context)
	RegisterHandlersWithOptions(r, a, GinServerOptions{BaseURL: a.BasePath})
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
	gr := NewGraph(g, params.Options)
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
			nodes = append(nodes, node(n, GraphOptions{}))
		}
	})
	okResponse(c, nodes)
}

func (a *API) GraphNeighbors(c *gin.Context, params GraphNeighborsParams) {
	e := a.Engine
	r := Neighbors{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return
	}
	start, err := TraverseStart(e, r.Start)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	ctx, cancel := korrel8r.WithConstraint(c.Request.Context(), r.Start.Constraint.Default())
	defer cancel()
	g, err := traverse.Neighbors(ctx, e, start, r.Depth)
	if !check(c, http.StatusNotFound, err) {
		return
	}
	gr := NewGraph(g, params.Options)
	okResponse(c, gr)
}

// GraphNeighbours alias for alternate spelling.
//
// Deprecated: Use GraphNeighbors, korrel8r now uses US spelling consistently.
func (a *API) GraphNeighbours(c *gin.Context, params GraphNeighboursParams) {
	a.GraphNeighbors(c, GraphNeighborsParams(params))
}

func (a *API) Objects(c *gin.Context, params ObjectsParams) {
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

// goals is shared between GraphGoals and ListGoals
func (a *API) goals(c *gin.Context) (*graph.Graph, []korrel8r.Class) {
	e := a.Engine
	r := Goals{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return nil, nil
	}
	start, err := TraverseStart(e, r.Start)
	if !check(c, http.StatusBadRequest, err) {
		return nil, nil
	}
	goals, err := e.Classes(([]string)(r.Goals))
	if !check(c, http.StatusBadRequest, err) {
		return nil, nil
	}
	ctx, cancel := korrel8r.WithConstraint(c.Request.Context(), r.Start.Constraint.Default())
	defer cancel()
	g, err := traverse.Goals(ctx, a.Engine, start, goals)
	check(c, http.StatusNotFound, err)
	return g, goals
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if c.IsAborted() {
		return false
	}
	if err == nil {
		return true
	}
	if len(format) > 0 {
		err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
	}
	c.AbortWithStatusJSON(code, c.Error(err).JSON())
	return false
}

// context sets up authorization and local deadline for outgoing requests.
func (a *API) context(c *gin.Context) {
	ctx := auth.Context(c.Request) // add authentication
	requestTimeout := ptr.To(time.Minute)
	if len(a.Configs) > 0 {
		tuning := a.Configs[0].Tuning
		if tuning != nil && tuning.RequestTimeout != nil {
			requestTimeout = &tuning.RequestTimeout.Duration
		}
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(*requestTimeout))
	c.Request = c.Request.WithContext(ctx)
	defer cancel()

	c.Next()
}

// okResponse sets an OK response with a body if we were not already aborted.
func okResponse(c *gin.Context, body any) {
	if !c.IsAborted() {
		c.JSON(http.StatusOK, body)
	}
}
