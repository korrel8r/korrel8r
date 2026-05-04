// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rest implements a REST API for korrel8r.
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/session"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

var log = logging.Log()

type API struct {
	Sessions session.Manager
	Router   *gin.Engine
}

// session returns the per-request Session from the context.
func (a *API) session(c *gin.Context) (*session.Session, error) {
	ctx := c.Request.Context()
	// First check for session already set on context by transport.
	if s := session.FromContext(ctx); s != nil {
		return s, nil
	}
	return a.Sessions.Get(ctx)
}

var _ ServerInterface = &API{}

// New API instance, registers handlers with a gin Engine.
func New(sessions session.Manager, r *gin.Engine) (*API, error) {
	a := &API{
		Sessions: sessions,
		Router:   r,
	}
	r.Use(session.Middleware(sessions))
	rg := r.Group(BasePath)
	rg.Use(a.logger) // Apply logger only to API endpoints
	RegisterHandlers(rg, a)
	// Helpful endpoints showing routes.
	r.GET(BasePath, func(c *gin.Context) { spec, _ := api.GetSwagger(); c.JSON(http.StatusOK, spec) })
	r.GET("/", a.homePage)
	return a, nil
}

func (a *API) homePage(c *gin.Context) {
	paths := unique.NewList[string]()
	for _, r := range a.Router.Routes() {
		paths.Add(r.Path)
	}
	slices.Sort(paths.List)
	c.String(http.StatusOK, fmt.Sprintf("Korrel8r %v\n\n%v", build.Version, strings.Join(paths.List, "\n")))
}

func (a *API) ListDomains(c *gin.Context) {
	session, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return
	}
	c.JSON(http.StatusOK, ListDomains(session.Engine))
}

func (a *API) ListDomainClasses(c *gin.Context, domain string) {
	session, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return
	}
	d, err := session.Engine.Domain(domain)
	if !check(c, http.StatusNotFound, err, "domain not found: %s", domain) {
		return
	}

	var classNames []string
	for _, class := range d.Classes() {
		classNames = append(classNames, class.Name())
	}
	c.JSON(http.StatusOK, classNames)
}

func (a *API) GraphGoals(c *gin.Context, params GraphGoalsParams) {
	g, _ := a.goals(c)
	gr := NewGraph(g, params.Options)
	okResponse(c, gr)
}

func (a *API) ListGoals(c *gin.Context) {
	nodes := []api.Node{} // return [] not null for empty
	g, goals := a.goals(c)
	if c.IsAborted() {
		return
	}
	set := unique.NewSet(goals...)
	g.EachNode(func(n *graph.Node) {
		if set.Has(n.Class) {
			nodes = append(nodes, node(n, api.GraphOptions{}))
		}
	})
	okResponse(c, nodes)
}

func (a *API) GraphNeighbors(c *gin.Context, params GraphNeighborsParams) {
	session, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return
	}
	e := session.Engine
	r := api.Neighbors{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return
	}
	start, err := TraverseStart(e, r.Start)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	g, err := traverse.Neighbors(c.Request.Context(), e, start, r.Depth)
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
	session, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return
	}
	e := session.Engine
	query, err := e.Query(params.Query)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	constraint := (*korrel8r.Constraint)(nil) // TODO can't pass constraints
	result := result.New(query.Class())
	if !check(c, http.StatusNotFound, e.Get(c.Request.Context(), query, constraint, result)) {
		return
	}
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

func (a *API) ReplaceStore(c *gin.Context, domain string) {
	var store Store
	if !check(c, http.StatusBadRequest, c.BindJSON(&store)) {
		return
	}

	// Ensure domain matches the path parameter
	store[config.StoreKeyDomain] = domain

	if !check(c, http.StatusBadRequest, a.Engine.ReplaceStore(config.Store(store))) {
		return
	}

	log.V(1).Info("Store replaced", "domain", domain, "config", store)
	c.JSON(http.StatusOK, gin.H{"status": "store replaced", "domain": domain})
}

// goals is shared between GraphGoals and ListGoals
func (a *API) goals(c *gin.Context) (*graph.Graph, []korrel8r.Class) {
	session, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return nil, nil
	}
	e := session.Engine
	r := api.Goals{}
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
	g, err := traverse.Goals(c.Request.Context(), e, start, goals)
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

// okResponse sets an OK response with a body if we were not already aborted.
func okResponse(c *gin.Context, body any) {
	if !c.IsAborted() {
		c.JSON(http.StatusOK, body)
	}
}

// Set the actual console display, called by the console itself.
// (POST /console)
func (a *API) SetConsole(c *gin.Context) {
	state := &api.Console{}
	if !check(c, http.StatusBadRequest, c.BindJSON(state)) {
		return
	}
	s, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return
	}
	if !check(c, http.StatusBadRequest, ConsoleOK(s.Engine, state)) {
		return
	}
	s.ConsoleState.Set(state)
	c.JSON(http.StatusOK, state)
}

// SSE notification of console updates.
// (GET /console/events)
func (a *API) ConsoleEvents(c *gin.Context) {
	// Set SSE headers
	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	w.Flush() // Send headers to start the SSE stream.

	s, err := a.session(c)
	if !check(c, http.StatusInternalServerError, err) {
		return
	}
	log := log // Don't modify main logger
	if s.ID != "" {
		log = log.WithValues("session", s)
	}
	keepAliveTicker := time.NewTicker(time.Minute)
	defer keepAliveTicker.Stop()
	log.V(3).Info("Console events started")
	defer log.V(3).Info("Console events stopped")

	state, next := s.ConsoleRequest.GetChan()
	if !check(c, http.StatusInternalServerError, a.sendEvent(w, state)) {
		return
	}
	for {
		select {
		case <-next: // Wait for an new value
			state, next = s.ConsoleRequest.GetChan()
			if !check(c, http.StatusInternalServerError, a.sendEvent(w, state)) {
				return
			}
		case <-keepAliveTicker.C:
			// Send a keep-alive comment
			_, err = fmt.Fprint(w, ":keepalive\n\n")
			if !check(c, http.StatusInternalServerError, err) {
				return
			}
			w.Flush()

		case <-c.Request.Context().Done():
			return
		}
	}
}

func (a *API) sendEvent(w gin.ResponseWriter, update *api.Console) error {
	if update != nil {
		b, _ := json.Marshal(update)
		_, err := fmt.Fprintf(w, "event: console-update\ndata: %v\n\n", string(b))
		if err != nil {
			return err
		}
		w.Flush()
		log.V(3).Info("Console event sent", "event", string(b))
	}
	return nil
}
