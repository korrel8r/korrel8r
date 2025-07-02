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
	r.Use(a.logger)
	r.Use(a.context)
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
	e := a.Engine
	r := Neighbours{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return
	}
	start, err := TraverseStart(e, r.Start)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	ctx, cancel := korrel8r.WithConstraint(c.Request.Context(), r.Start.Constraint.Default())
	defer cancel()
	g, err := traverse.New(e, e.Graph()).Neighbours(ctx, start, r.Depth)
	if !check(c, http.StatusNotFound, err) {
		return
	}
	gr := Graph{Nodes: nodes(g), Edges: edges(g, ptr.Deref(params.Rules))}
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
	g := e.Graph().ShortestPaths(start.Class, goals...)
	ctx, cancel := korrel8r.WithConstraint(c.Request.Context(), r.Start.Constraint.Default())
	defer cancel()
	g, err = traverse.New(a.Engine, g).Goals(ctx, start, goals)
	check(c, http.StatusNotFound, err)
	return g, goals
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if err != nil && len(format) > 0 {
		err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
	}
	switch {
	case c.IsAborted():
		return false
	case err == nil:
		return true
	case traverse.IsPartialError(err):
		_ = c.Error(err) // Save the error for interrupted()
		return true      // Allow the handler to succeed
	default:
		ginErr := c.Error(err)
		c.AbortWithStatusJSON(code, ginErr.JSON())
		log.Error(err, "Request failed", "url", c.Request.URL, "code", code, "error", err)
		return false
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

// Interrupted checks for a context error or partial error,
// indicating we may have a partial result.
func interrupted(c *gin.Context) bool {
	return c.Errors.Last() != nil && traverse.IsPartialError(c.Errors.Last()) ||
		c.Request.Context().Err() == context.DeadlineExceeded
}

// okResponse sets an OK response with a body if we were not already aborted.
func okResponse(c *gin.Context, body any) {
	if !c.IsAborted() {
		status := http.StatusOK
		if interrupted(c) {
			status = http.StatusPartialContent
		}
		c.JSON(status, body)
	}
}
