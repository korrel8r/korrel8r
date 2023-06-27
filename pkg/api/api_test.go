// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package api

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ginEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	if flag.Lookup("test.v") != nil {
		gin.SetMode(gin.DebugMode)
	}
	r := gin.New()
	if flag.Lookup("test.v") != nil {
		r.Use(gin.Logger())
	}
	return r
}

func testAPI(domains ...korrel8r.Domain) *API {
	return test.Must(New(engine.New(domains...), ginEngine()))
}

func do(a *API, method, url string, body io.Reader) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		w.Code = http.StatusInternalServerError
		fmt.Fprintln(w, err.Error())
	}
	a.Router.ServeHTTP(w, req)
	return w
}

func assertDo[T any](t *testing.T, a *API, method, url string, req any, code int, want T) {
	t.Helper()
	var r io.Reader
	if req != nil {
		r = strings.NewReader(test.JSONString(req))
	}
	w := do(a, method, url, r)
	if assert.Equal(t, code, w.Code, w.Body.String()) {
		var got T
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got), w.Body.String())
		assert.Equal(t, want, got)
	}
}

func TestAPI_GET_domains(t *testing.T) {
	assertDo(t, testAPI(mock.Domains("foo", "bar")...), "GET", "/api/v1alpha1/domains", nil, 200, Domains{"foo", "bar"})
}

func TestAPI_GET_stores(t *testing.T) {
	a := testAPI(mock.Domains("foo", "bar")...)
	require.NoError(t, a.Engine.AddStoreConfig(korrel8r.StoreConfig{"domain": "foo", "a": "1"}))
	require.NoError(t, a.Engine.AddStoreConfig(korrel8r.StoreConfig{"domain": "foo", "b": "2"}))
	require.NoError(t, a.Engine.AddStoreConfig(korrel8r.StoreConfig{"domain": "bar", "x": "y"}))

	assertDo(t, a, "GET", "/api/v1alpha1/stores/foo", nil, 200, Stores{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}})
	assertDo(t, a, "GET", "/api/v1alpha1/stores/bar", nil, 200, Stores{{"domain": "bar", "x": "y"}})
	assertDo(t, a, "GET", "/api/v1alpha1/stores", nil, 200, Stores{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}, {"domain": "bar", "x": "y"}})
	assertDo(t, a, "GET", "/api/v1alpha1/stores/bad", nil, 404, gin.H{"error": `domain not found: "bad"`})
}

func doubleFunc(goal korrel8r.Class) func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error) {
	return func(o korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
		return mock.NewQuery(goal, o.(string)+o.(string)), nil
	}
}

func apiWithRules() (a *API, x, y, z korrel8r.Class) {
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	api := testAPI(foo, bar)
	x, y, z = foo.Class("x"), bar.Class("y"), bar.Class("z")
	test.PanicErr(api.Engine.AddStore(mock.NewStore(foo)))
	test.PanicErr(api.Engine.AddStore(mock.NewStore(bar)))
	api.Engine.AddRules(mock.NewApplyRule(x, y, doubleFunc(y)))
	api.Engine.AddRules(mock.NewQueryRule(y, mock.NewQuery(z, "c")))
	return api, x, y, z
}

func TestAPI_GET_goals(t *testing.T) {
	a, x, y, _ := apiWithRules()
	assertDo(t, a, "GET", "/api/v1alpha1/goals?start=foo+x&goal=bar+y&query="+url.QueryEscape(mock.NewQuery(x, "a").String()), nil,
		200, Results{{Class: newClass(y), Queries: QueryCounts{mock.NewQuery(y, "aa").String(): 1}}})
}

func TestAPI_POST_goals(t *testing.T) {
	a, x, y, _ := apiWithRules()
	assertDo(t, a, "POST", "/api/v1alpha1/goals",
		GoalsRequest{
			Start: Start{
				Start:   newClass(x),
				Query:   mock.NewQuery(x, "a").String(),
				Objects: []json.RawMessage{[]byte(`"b"`)},
			},
			Goals: []Class{newClass(y)},
		},
		200, Results{{Class: Class{Class: "y", Domain: "bar"}, Queries: QueryCounts{
			mock.NewQuery(y, "aa").String(): 1,
			mock.NewQuery(y, "bb").String(): 1,
		}}})
}

func TestAPI_GET_graphs(t *testing.T) {
	a, x, y, _ := apiWithRules()
	assertDo(t, a, "GET",
		"/api/v1alpha1/graphs?start=foo+x&goal=bar+y&query="+url.QueryEscape(mock.NewQuery(x, "a").String()), nil,
		200,
		Graph{
			Nodes: []Result{
				{Class: newClass(x)},
				{Class: newClass(y), Queries: QueryCounts{mock.NewQuery(y, "aa").String(): 1}},
			},
			Edges: [][2]int{{0, 1}},
		})
}

func TestAPI_POST_graphs(t *testing.T) {
	a, x, y, z := apiWithRules()
	assertDo(t, a, "POST", "/api/v1alpha1/graphs",
		GoalsRequest{
			Start: Start{
				Start:   newClass(x),
				Query:   mock.NewQuery(x, "a").String(),
				Objects: []json.RawMessage{[]byte(`"b"`)},
			},
			Goals: []Class{newClass(z)},
		},
		200,
		Graph{
			Nodes: []Result{
				{Class: newClass(x)},
				{Class: newClass(y), Queries: QueryCounts{
					mock.NewQuery(y, "aa").String(): 1,
					mock.NewQuery(y, "bb").String(): 1,
				}},
				{Class: newClass(z), Queries: QueryCounts{mock.NewQuery(z, "c").String(): 1}},
			},
			Edges: [][2]int{{0, 1}, {1, 2}},
		})
}

func TestAPI_GET_neighbours(t *testing.T) {
	a, x, y, _ := apiWithRules()
	assertDo(t, a, "GET", "/api/v1alpha1/neighbours?start=foo+x&depth=1&query="+mock.NewQuery(x, "a").String(), nil,
		200,
		Graph{
			Nodes: []Result{
				{Class: newClass(x)},
				{Class: newClass(y), Queries: QueryCounts{mock.NewQuery(y, "aa").String(): 1}}},
			Edges: [][2]int{{0, 1}},
		})
}

func TestAPI_POST_neighbours(t *testing.T) {
	a, x, y, z := apiWithRules()
	assertDo(t, a, "POST", "/api/v1alpha1/neighbours",
		NeighboursRequest{
			Start: Start{
				Start:   newClass(x),
				Objects: []json.RawMessage{[]byte(`"a"`)},
			},
			Depth: 2,
		},
		200,
		Graph{
			Nodes: []Result{
				{Class: newClass(x)},
				{Class: newClass(y), Queries: QueryCounts{mock.NewQuery(y, "aa").String(): 1}},
				{Class: newClass(z), Queries: QueryCounts{mock.NewQuery(z, "c").String(): 1}},
			},
			Edges: [][2]int{{0, 1}, {1, 2}},
		})
}
