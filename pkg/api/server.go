// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

type API struct {
	Engine *engine.Engine
	Router *gin.Engine
}

// New API instance, registers with gin Engine.
func New(e *engine.Engine, r *gin.Engine) (*API, error) {
	a := &API{Engine: e, Router: r}
	v := r.Group("/api/v1alpha1")
	v.GET("/domains", a.getDomains)
	v.GET("/stores/:domain", a.getStoresDomain)
	v.GET("/stores", a.getStores)
	v.POST("/goals", a.postGoals)
	v.GET("/goals", a.getGoals)
	v.GET("/graphs", a.getGraphs)
	v.POST("/graphs", a.postGraphs)
	v.GET("/neighbours", a.getNeighbours)
	v.POST("/neighbours", a.postNeighbours)
	return a, nil
}

// Close cleans any persistent resources.
func (a *API) Close() {}

func (a *API) getDomains(c *gin.Context) {
	var domains []string
	for _, d := range a.Engine.Domains() {
		domains = append(domains, d.String())
	}
	c.JSON(http.StatusOK, domains)
}

func (a *API) getStores(c *gin.Context) {
	var stores []korrel8r.StoreConfig
	for _, d := range a.Engine.Domains() {
		stores = append(stores, a.Engine.StoreConfigsFor(d)...)
	}
	c.JSON(http.StatusOK, stores)
}

func (a *API) getStoresDomain(c *gin.Context) {
	name := strings.TrimPrefix(c.Params.ByName("domain"), "/")
	d, err := a.Engine.DomainErr(name)
	if check(c, http.StatusNotFound, err) {
		c.JSON(http.StatusOK, a.Engine.StoreConfigsFor(d))
	}
}

func (a *API) getGoals(c *gin.Context) {
	start, query := a.startParams(c)
	goals := a.goalsParams(c)
	a.goalsResponse(c, start, nil, query, goals)
}

func (a *API) postGoals(c *gin.Context) {
	start, goals, query, objects := a.goalsPostRequest(c)
	if ok(c) {
		a.goalsResponse(c, start, objects, query, goals)
	}
}

func (a *API) getGraphs(c *gin.Context) {
	start, query := a.startParams(c)
	goals := a.goalsParams(c)
	a.graphsResponse(c, start, nil, query, goals)
}
func (a *API) postGraphs(c *gin.Context) {
	start, goals, query, objects := a.goalsPostRequest(c)
	a.graphsResponse(c, start, objects, query, goals)
}

func (a *API) getNeighbours(c *gin.Context) {
	start, query := a.startParams(c)
	depth, err := strconv.Atoi(c.Query("depth"))
	if check(c, http.StatusBadRequest, err, "depth parameter") {
		a.neighboursResponse(c, start, nil, query, depth)
	}
}

func (a *API) postNeighbours(c *gin.Context) {
	var req NeighboursRequest
	if check(c, http.StatusBadRequest, c.BindJSON(&req), "NeighboursRequest body") {
		start, query, objects := a.startRequest(c, &req.Start)
		if ok(c) {
			a.neighboursResponse(c, start, objects, query, req.Depth)
		}
	}
}

func (a *API) startParams(c *gin.Context) (start korrel8r.Class, query korrel8r.Query) {
	start = a.queryClass(c, "start")
	var err error
	query, err = start.Domain().Query(c.Query("query"))
	check(c, http.StatusBadRequest, err, "query parameter")
	return start, query
}

func (a *API) goalsParams(c *gin.Context) (goals []korrel8r.Class) {
	goals = a.queryClasses(c, "goal")
	return goals
}

func (a *API) goalsPostRequest(c *gin.Context) (start korrel8r.Class, goals []korrel8r.Class, query korrel8r.Query, objects []korrel8r.Object) {
	var req GoalsRequest
	if !check(c, http.StatusBadRequest, c.BindJSON(&req), "GoalsRequest body") {
		return
	}
	start, query, objects = a.startRequest(c, &req.Start)
	goals = a.classes(c, req.Goals)
	return
}

func (a *API) startRequest(c *gin.Context, s *Start) (start korrel8r.Class, query korrel8r.Query, objects []korrel8r.Object) {
	start = a.class(c, s.Start)
	var err error
	if s.Query != "" {
		query, err = start.Domain().Query(s.Query)
		check(c, http.StatusBadRequest, err, "query field")
	}
	for _, raw := range s.Objects {
		obj := start.New()
		if check(c, http.StatusBadRequest, json.Unmarshal(raw, &obj), "objects field") {
			objects = append(objects, obj)
		}
	}
	return start, query, objects
}

func (a *API) goalsGraph(c *gin.Context, start korrel8r.Class, from []korrel8r.Object, query korrel8r.Query, goals []korrel8r.Class) *graph.Graph {
	g := a.Engine.Graph().AllPaths(start, goals...)
	ctx := c.Request.Context()
	if check(c, http.StatusInternalServerError, a.setupStart(ctx, g, start, from, query)) &&
		check(c, http.StatusInternalServerError, g.Traverse(a.Engine.Follower(ctx).Traverse)) {
		return g
	}
	return nil
}

func (a *API) goalsResponse(c *gin.Context, start korrel8r.Class, from []korrel8r.Object, query korrel8r.Query, goals []korrel8r.Class) {
	if ok(c) {
		g := a.goalsGraph(c, start, from, query, goals)
		if ok(c) {
			c.JSON(http.StatusOK, nodeResults(g, goals...))
		}
	}
}

func (a *API) graphsResponse(c *gin.Context, start korrel8r.Class, from []korrel8r.Object, query korrel8r.Query, goals []korrel8r.Class) {
	if ok(c) {
		g := a.goalsGraph(c, start, from, query, goals)
		if ok(c) {
			c.JSON(http.StatusOK, newGraph(g))
		}
	}
}

func (a *API) neighboursResponse(c *gin.Context, start korrel8r.Class, from []korrel8r.Object, query korrel8r.Query, depth int) {
	g := a.Engine.Graph()
	ctx := c.Request.Context()
	if check(c, http.StatusInternalServerError, a.setupStart(ctx, g, start, from, query)) {
		g := g.Neighbours(start, depth, a.Engine.Follower(ctx).Traverse)
		c.JSON(http.StatusOK, newGraph(g))
	}
}

func (a *API) setupStart(ctx context.Context, g *graph.Graph, start korrel8r.Class, from []korrel8r.Object, query korrel8r.Query) error {
	result := g.NodeFor(start).Result
	result.Append(from...)
	if query != nil {
		return a.Engine.Get(ctx, start, query, result)
	}
	return nil
}

func (a *API) queryClass(c *gin.Context, param string) korrel8r.Class {
	apiClass := Class{}
	apiClass.Domain, apiClass.Class, _ = strings.Cut(c.Query(param), " ")
	return a.class(c, apiClass)
}

func (a *API) queryClasses(c *gin.Context, param string) (classes []korrel8r.Class) {
	names := c.QueryArray(param)
	for _, name := range names {
		apiClass := Class{}
		apiClass.Domain, apiClass.Class, _ = strings.Cut(name, " ")
		if class := a.class(c, apiClass); class != nil {
			classes = append(classes, class)
		}
	}
	return classes
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
		}
		ginErr := c.Error(err)
		c.AbortWithStatusJSON(code, ginErr.JSON())
	}
	return err == nil
}

func (a *API) class(c *gin.Context, apiClass Class) korrel8r.Class {
	class, err := a.Engine.DomainClass(apiClass.Domain, apiClass.Class)
	check(c, http.StatusNotFound, err)
	return class
}

func (a *API) classes(c *gin.Context, apiClasses []Class) (classes []korrel8r.Class) {
	for _, apiClass := range apiClasses {
		if class := a.class(c, apiClass); class != nil {
			classes = append(classes, class)
		}
	}
	return classes
}

func newClass(c korrel8r.Class) Class { return Class{Domain: c.Domain().String(), Class: c.String()} }

func newQueryCounts(qcs graph.QueryCounts) QueryCounts {
	if len(qcs) == 0 {
		return nil
	}
	ret := make(QueryCounts, len(qcs))
	for _, qc := range qcs.Sort() {
		ret[qc.Query.String()] = qc.Count
	}
	return ret
}

func nodeResult(n *graph.Node) Result {
	return Result{Class: newClass(n.Class), Queries: newQueryCounts(n.QueryCounts)}
}

func nodeResults(g *graph.Graph, classes ...korrel8r.Class) (results []Result) {
	for _, c := range classes {
		results = append(results, nodeResult(g.NodeFor(c)))
	}
	return results
}

func newGraph(g *graph.Graph) *Graph {
	apiGraph := &Graph{}
	ids := map[int64]int{}
	node := func(n *graph.Node) int {
		if _, ok := ids[n.ID()]; !ok {
			ids[n.ID()] = len(apiGraph.Nodes)
			apiGraph.Nodes = append(apiGraph.Nodes, nodeResult(n))
		}
		return ids[n.ID()]
	}
	edges := g.Edges()
	for edges.Next() {
		e := edges.Edge()
		u := node(e.From().(*graph.Node))
		v := node(e.To().(*graph.Node))
		apiGraph.Edges = append(apiGraph.Edges, [2]int{u, v})
	}
	return apiGraph
}

func ok(c *gin.Context) bool { return len(c.Errors) == 0 }
