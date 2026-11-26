// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rest implements a REST API for korrel8r.
package rest

import (
	"context"
	// Request bodies use encoding/json directly: its syntax-error messages are part of the API.
	stdjson "encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/json"
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

var _ ServerInterface = (*API)(nil)

type API struct {
	Sessions session.Manager
	Router   *http.ServeMux
	paths    []string
}

// session returns the per-request Session from the context.
func (a *API) session(r *http.Request) (*session.Session, error) {
	ctx := r.Context()
	// First check for session already set on context by transport.
	if s := session.FromContext(ctx); s != nil {
		return s, nil
	}
	return a.Sessions.Get(ctx)
}

// New registers the REST API on a standard HTTP mux.
func New(sessions session.Manager, r *http.ServeMux) (*API, error) {
	a := &API{Sessions: sessions, Router: r}
	HandlerWithOptions(a, StdHTTPServerOptions{
		BaseURL:    api.BasePath,
		BaseRouter: &apiMux{mux: r, api: a, sessions: sessions},
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeJSON(w, http.StatusBadRequest, api.Error{Error: err.Error()})
		},
	})
	r.Handle("GET "+api.BasePath, Recover(session.Middleware(sessions)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spec, _ := api.GetSpec()
		writeJSON(w, http.StatusOK, spec)
	}))))
	r.Handle("GET /{$}", Recover(session.Middleware(sessions)(http.HandlerFunc(a.homePage))))
	a.AddPaths(api.BasePath, "/")
	return a, nil
}

// AddPaths adds paths served by other handlers on the same mux to the home page listing.
func (a *API) AddPaths(paths ...string) { a.paths = append(a.paths, paths...) }

func (a *API) homePage(w http.ResponseWriter, _ *http.Request) {
	paths := unique.NewList[string]()
	for _, path := range a.paths {
		paths.Add(path)
	}
	slices.Sort(paths.List)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "Korrel8r %v\n\n%v", build.Version, strings.Join(paths.List, "\n"))
}

func (a *API) ListDomains(w http.ResponseWriter, r *http.Request) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	writeJSON(w, http.StatusOK, ListDomains(session.Engine))
}

func (a *API) ListDomainClasses(w http.ResponseWriter, r *http.Request, domain string) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	d, err := session.Engine.Domain(domain)
	if !check(w, http.StatusNotFound, err) {
		return
	}

	var classNames []string
	for _, class := range d.Classes() {
		classNames = append(classNames, class.Name())
	}
	writeJSON(w, http.StatusOK, classNames)
}

func (a *API) GraphGoals(w http.ResponseWriter, r *http.Request, params GraphGoalsParams) {
	g, _, ok := a.goals(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, NewGraph(g, params.Options))
}

func (a *API) ListGoals(w http.ResponseWriter, r *http.Request) {
	nodes := []api.Node{} // return [] not null for empty
	g, goals, ok := a.goals(w, r)
	if !ok {
		return
	}
	set := unique.NewSet(goals...)
	g.EachNode(func(n *graph.Node) {
		if set.Has(n.Class) {
			nodes = append(nodes, node(n, api.GraphOptions{}))
		}
	})
	writeJSON(w, http.StatusOK, nodes)
}

func (a *API) GraphNeighbors(w http.ResponseWriter, r *http.Request, params GraphNeighborsParams) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	e := session.Engine
	input := api.Neighbors{}
	if !check(w, http.StatusBadRequest, stdjson.NewDecoder(r.Body).Decode(&input)) {
		return
	}
	start, err := TraverseStart(e, input.Start)
	if !check(w, http.StatusBadRequest, err) {
		return
	}
	g, err := traverse.Neighbors(r.Context(), e, start, input.Depth)
	if !checkTraversal(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, NewGraph(g, params.Options))
}

// GraphNeighbours alias for alternate spelling.
//
// Deprecated: Use GraphNeighbors, korrel8r now uses US spelling consistently.
func (a *API) GraphNeighbours(w http.ResponseWriter, r *http.Request, params GraphNeighboursParams) {
	a.GraphNeighbors(w, r, GraphNeighborsParams(params))
}

func (a *API) Objects(w http.ResponseWriter, r *http.Request, params ObjectsParams) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	e := session.Engine
	query, err := e.Query(params.Query)
	if !check(w, http.StatusBadRequest, err) {
		return
	}
	constraint := Constraint(params.Constraint)
	result := result.New(query.Class())
	if !check(w, http.StatusNotFound, e.Get(r.Context(), query, constraint, result)) {
		return
	}
	body := []any(result.List())
	if body == nil {
		body = []any{} // Return [] on empty, not null.
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *API) SetConfig(w http.ResponseWriter, _ *http.Request, params SetConfigParams) {
	if params.Verbose != nil {
		log.V(1).Info("Config set verbose", "level", *params.Verbose)
		logging.SetVerbose(*params.Verbose)
	}
	writeJSON(w, http.StatusOK, params)
}

// goals is shared between GraphGoals and ListGoals. Errors are written before returning ok=false.
func (a *API) goals(w http.ResponseWriter, r *http.Request) (*graph.Graph, []korrel8r.Class, bool) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return nil, nil, false
	}
	e := session.Engine
	input := api.Goals{}
	if !check(w, http.StatusBadRequest, stdjson.NewDecoder(r.Body).Decode(&input)) {
		return nil, nil, false
	}
	start, err := TraverseStart(e, input.Start)
	if !check(w, http.StatusBadRequest, err) {
		return nil, nil, false
	}
	goals, err := e.Classes(([]string)(input.Goals))
	if !check(w, http.StatusBadRequest, err) {
		return nil, nil, false
	}
	g, err := traverse.Goals(r.Context(), e, start, goals)
	return g, goals, checkTraversal(w, err)
}

func checkTraversal(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	if limitErr, ok := errors.AsType[*traverse.LimitError](err); ok {
		// Truncation is a successful partial result. Headers also expose the
		// condition for endpoints whose response body is not a graph.
		w.Header().Set("X-Korrel8r-Truncated", "true")
		w.Header().Set("X-Korrel8r-Truncated-By", limitErr.Name)
		w.Header().Set("X-Korrel8r-Truncated-Limit", fmt.Sprint(limitErr.Limit))
		return true
	}
	return check(w, http.StatusNotFound, err)
}

// check writes a JSON error response and returns false if err is non-nil.
func check(w http.ResponseWriter, code int, err error) bool {
	if err == nil {
		return true
	}
	writeJSON(w, code, api.Error{Error: err.Error()})
	return false
}

func (a *API) Help(w http.ResponseWriter, r *http.Request) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	doc, err := DomainHelp(session.Engine, "")
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	writeJSON(w, http.StatusOK, api.Help{Documentation: doc})
}

func (a *API) HelpDomain(w http.ResponseWriter, r *http.Request, domain string) {
	session, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	doc, err := DomainHelp(session.Engine, domain)
	if !check(w, http.StatusNotFound, err) {
		return
	}
	writeJSON(w, http.StatusOK, api.Help{Documentation: doc})
}

// GetConsole returns the current console state.
// (GET /console)
func (a *API) GetConsole(w http.ResponseWriter, r *http.Request) {
	s, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	state := s.ConsoleState()
	if state == nil {
		check(w, http.StatusNotFound, session.ErrNoConsole)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// ShowInConsole pushes a console display update to the SSE stream.
// (PUT /console/events)
func (a *API) ShowInConsole(w http.ResponseWriter, r *http.Request) {
	update := &api.Console{}
	if !check(w, http.StatusBadRequest, stdjson.NewDecoder(r.Body).Decode(update)) {
		return
	}
	s, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	if !check(w, http.StatusBadRequest, ConsoleOK(s.Engine, update)) {
		return
	}
	if !check(w, http.StatusNotFound, s.ShowInConsole(update)) {
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// Set the actual console display, called by the console itself.
// (PUT /console)
func (a *API) SetConsole(w http.ResponseWriter, r *http.Request) {
	state := &api.Console{}
	if !check(w, http.StatusBadRequest, stdjson.NewDecoder(r.Body).Decode(state)) {
		return
	}
	s, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}
	if !check(w, http.StatusBadRequest, ConsoleOK(s.Engine, state)) {
		return
	}
	s.SetConsoleState(state)
	writeJSON(w, http.StatusOK, state)
}

// SSE notification of console updates.
// (GET /console/events)
func (a *API) ConsoleEvents(w http.ResponseWriter, r *http.Request) {
	s, err := a.session(r)
	if !check(w, http.StatusInternalServerError, err) {
		return
	}

	// Strip the middleware's request timeout — SSE is long-lived.
	// context.WithoutCancel preserves values but detaches from the deadline,
	// then AfterFunc re-attaches disconnect detection from the parent.
	ctx, cancel := context.WithCancel(context.WithoutCancel(r.Context()))
	defer cancel()
	stop := context.AfterFunc(r.Context(), cancel)
	defer stop()

	err = s.ConsoleEvents(
		ctx,
		func() error {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
			flush(w)
			return nil
		},
		func(c *api.Console) error { return a.sendEvent(w, c) },
		func() error { _, err := fmt.Fprint(w, ":keepalive\n\n"); flush(w); return err },
		3*time.Second)
	if errors.Is(err, session.ErrConsoleBusy) {
		// Reported before the event stream starts, so no response has been written yet.
		check(w, http.StatusConflict, err)
		return
	}
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		log.V(3).Error(err, "Console SSE error", "session", s)
	}
}

func (a *API) sendEvent(w http.ResponseWriter, update *api.Console) error {
	b, _ := json.Marshal(update)
	if _, err := w.Write([]byte("event: console-update\ndata: ")); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}
	flush(w)
	log.V(3).Info("Console update sent", "event", string(b))
	return nil
}
