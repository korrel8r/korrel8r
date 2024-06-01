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
//	@schemes		http https
//	@accept			json
//	@produce		json
package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rest/docs"
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
}

// New API instance, registers  handlers with a gin Engine.
func New(e *engine.Engine, c config.Configs, r *gin.Engine) (*API, error) {
	a := &API{Engine: e, Configs: c}
	r.Use(a.logRequest)
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html") })
	r.GET("/api", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html") })
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggofiles.Handler))
	v := r.Group(docs.SwaggerInfo.BasePath)
	v.GET("/domains", a.Domains)
	v.GET("/domains/:domain/classes", a.DomainClasses)
	v.POST("/graphs/goals", a.GraphsGoals)
	v.POST("/lists/goals", a.ListsGoals)
	v.POST("/graphs/neighbours", a.GraphsNeighbours)
	v.GET("/objects", a.GetObjects)
	return a, nil
}

// Close cleans any persistent resources.
func (a *API) Close() {}

// Domains handler
//
//	@router		/domains [get]
//	@summary	Get name, configuration and status for each domain.
//	@success	200		{array}		Domain
//	@failure	default	{string}	string
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

// DomainClasses handler
//
//	@router		/domains/{domain}/classes [get]
//	@summary	Get class names and descriptions for a domain.
//	@param		domain	path		string	true	"Domain name"
//	@success	200		{object}	Classes
//	@failure	default	{string}	string
func (a *API) DomainClasses(c *gin.Context) {
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
//	@param		rules	query		bool	false	"include rules in graph edges"
//	@param		request	body		Goals	true	"search from start to goal classes"
//	@success	200		{object}	Graph
//	@failure	default	{string}	string
func (a *API) GraphsGoals(c *gin.Context) {
	opts := &Options{}
	if check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		if g, _ := a.goals(c); g != nil {
			c.JSON(http.StatusOK, Graph{Nodes: nodes(g), Edges: edges(g, opts)})
		}
	}
}

// ListsGoals handler.
//
//	@router		/lists/goals [post]
//	@summary	Create a list of goal nodes related to a starting point.
//	@param		request	body		Goals	true	"search from start to goal classes"
//	@success	200		{array}		Node
//	@failure	default	{string}	string
func (a *API) ListsGoals(c *gin.Context) {
	nodes := []Node{} // return [] not null for empty
	g, goals := a.goals(c)
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
//	@summary	Create a neighbourhood graph around a start object to a given depth.
//	@param		rules	query		bool		false	"include rules in graph edges"
//	@param		request	body		Neighbours	true	"search from neighbours"
//	@success	200		{object}	Graph
//	@failure	default	{string}	string
func (a *API) GraphsNeighbours(c *gin.Context) {
	r, opts := Neighbours{}, Options{}
	if !(check(c, http.StatusBadRequest, c.BindJSON(&r)) && check(c, http.StatusBadRequest, c.BindUri(&opts))) {
		return
	}
	start, objects, queries, constraint := a.start(c, &r.Start)
	depth := r.Depth
	if c.Errors != nil {
		return
	}
	g, err := a.Engine.Neighbours(c.Request.Context(), start, objects, queries, constraint, depth)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	c.JSON(http.StatusOK, Graph{Nodes: nodes(g), Edges: edges(g, &opts)})
}

// GetObjects handler
//
// swagger:route GET		/objects
//
//	@summary	Execute a query, returns a list of JSON objects.
//	@param		query	query		string	true	"query string"
//	@success	200		{array}		any
//	@failure	default	{string}	string
func (a *API) GetObjects(c *gin.Context) {
	opts := &Objects{}
	if !check(c, http.StatusBadRequest, c.BindQuery(opts)) {
		return
	}
	query, err := a.Engine.Query(opts.Query)
	if !check(c, http.StatusBadRequest, err) {
		return
	}
	result := korrel8r.NewResult(query.Class())
	if !check(c, http.StatusInternalServerError, a.Engine.Get(c.Request.Context(), query, (*korrel8r.Constraint)(opts.Constraint), result)) {
		return
	}
	c.JSON(http.StatusOK, result.List())
}

func (a *API) goals(c *gin.Context) (g *graph.Graph, goals []korrel8r.Class) {
	r := Goals{}
	if !check(c, http.StatusBadRequest, c.BindJSON(&r)) {
		return nil, nil
	}
	start, objects, queries, constraint := a.start(c, &r.Start)
	goals = a.classes(c, r.Goals)
	if c.Errors != nil {
		return nil, nil
	}
	g = a.Engine.Graph().ShortestPaths(start, goals...)
	var err error
	g, err = a.Engine.GoalSearch(c.Request.Context(), g, start, objects, queries, constraint, goals)
	if !check(c, http.StatusInternalServerError, err) {
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
		if !check(c, http.StatusBadRequest, json.Unmarshal([]byte(r), &obj), "decoding object of class %v", class.String()) {
			return nil
		}
		objects = append(objects, obj)
	}
	return objects
}

// start validates and extracts data from the Start part of a request.
func (a *API) start(c *gin.Context, start *Start) (korrel8r.Class, []korrel8r.Object, []korrel8r.Query, *korrel8r.Constraint) {
	queries := a.queries(c, start.Queries)
	var class korrel8r.Class
	if start.Class == "" && len(queries) > 0 {
		class = queries[0].Class()
	} else {
		class = a.class(c, start.Class)
	}
	objects := a.objects(c, class, start.Objects)
	if c.Errors != nil {
		return nil, nil, nil, nil
	}
	return class, objects, queries, start.Constraint
}

func check(c *gin.Context, code int, err error, format ...any) (ok bool) {
	if err != nil {
		if len(format) > 0 {
			err = fmt.Errorf("%v: %w", fmt.Sprintf(format[0].(string), format[1:]...), err)
		}
		_ = c.Error(err)
		c.AbortWithStatusJSON(code, c.Errors)
		log.Error(err, "abort request", "url", c.Request.URL, "code", code, "errors", c.Errors)
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

func (a *API) logRequest(c *gin.Context) {
	if log.V(2).Enabled() {
		var body []byte
		if c.Request != nil && c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		log.V(2).Info("request received", "from", c.Request.RemoteAddr, "method", c.Request.Method, "url", c.Request.URL, "body", string(body))
	}
	c.Next()
}
