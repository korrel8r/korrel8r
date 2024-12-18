// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	logDomain "github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_GetDomains(t *testing.T) {
	e, err := engine.Build().
		Domains(mock.Domains("foo", "bar")...).
		StoreConfigs(
			config.Store{"domain": "foo", "a": "1"},
			config.Store{"domain": "foo", "b": "2"},
			config.Store{"domain": "bar", "x": "y"},
		).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	assertDo(t, a, "GET", "/api/v1alpha1/domains", nil, 200, []Domain{
		{Name: "bar", Stores: []config.Store{{"domain": "bar", "x": "y"}}},
		{Name: "foo", Stores: []config.Store{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}}},
	})
}

func TestAPI_GetDomainClasses(t *testing.T) {
	e, err := engine.Build().Domains(logDomain.Domain, metric.Domain).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	assertDo(t, a, "GET", "/api/v1alpha1/domains/log/classes", nil, 200, Classes{
		"application":    logDomain.Application.Description(),
		"audit":          logDomain.Audit.Description(),
		"infrastructure": logDomain.Infrastructure.Description(),
	})
	assertDo(t, a, "GET", "/api/v1alpha1/domains/metric/classes", nil, 200, Classes{
		"metric": metric.Domain.Classes()[0].Description(),
	})
}

func TestAPI_ListGoals(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/lists/goals",
		Goals{
			Start: Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Goals: []string{"mock:b"},
		},
		200, []Node{
			{
				Class:   "mock:b",
				Count:   1,
				Queries: []QueryCount{{Query: "mock:b:y", Count: 1}},
			},
		})
}

func TestAPI_GraphGoals_rules(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/goals?rules=true",
		Goals{
			Start: Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Goals: []string{"mock:b"},
		},
		200,
		Graph{
			Nodes: []Node{
				{
					Class: "mock:a",
					Count: 1,
				},
				{
					Class:   "mock:b",
					Count:   1,
					Queries: []QueryCount{{Query: "mock:b:y", Count: 1}},
				}},
			Edges: []Edge{{
				Start: "mock:a",
				Goal:  "mock:b",
				Rules: []Rule{{
					Name:    "a-b",
					Queries: []QueryCount{{Query: "mock:b:y", Count: 1}},
				}},
			}},
		})
}

func TestAPI_PostNeighbours(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbours{
			Start: Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Depth: 1,
		},
		200,
		Graph{
			Nodes: []Node{
				{
					Class: "mock:a",
					Count: 1,
				},
				{
					Class:   "mock:b",
					Count:   1,
					Queries: []QueryCount{{Query: "mock:b:y", Count: 1}},
				}},
			Edges: []Edge{{Start: "mock:a", Goal: "mock:b"}},
		},
	)
}

func TestAPI_PostNeighbours_empty(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbours{
			Start: Start{Queries: []string{"mock:a:nossuchthing"}},
			Depth: 1,
		},
		200, Graph{},
	)
}

func TestAPI_PostNeighbours_none(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbours{
			Start: Start{Queries: []string{"mock:b:y"}},
			Depth: 0,
		},
		200,
		Graph{
			Nodes: []Node{{
				Class:   "mock:b",
				Queries: []QueryCount{{Query: "mock:b:y", Count: 1}},
				Count:   1,
			}}},
	)
}

func TestAPI_GetObjects(t *testing.T) {
	want := []any{"a", "b", "c"}
	d := mock.Domain("x")
	c := d.Class("y")
	s := mock.NewStore(d)
	q := mock.NewQuery(c, "test")
	s.AddQuery(q, want)
	e, err := engine.Build().Domains(d).Stores(s).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	assertDo(t, a, "GET", "/api/v1alpha1/objects?query="+url.QueryEscape(q.String()), nil, 200, want)
}

func TestAPI_GetObjects_empty(t *testing.T) {
	d := mock.Domain("x")
	c := d.Class("y")
	q := mock.NewQuery(c, "test")
	s := mock.NewStore(d)
	e, err := engine.Build().Domains(d).Stores(s).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := do(t, a, "GET", "/api/v1alpha1/objects?query="+url.QueryEscape(q.String()), nil)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "[]", w.Body.String())
}

func ginEngine() *gin.Engine {
	if os.Getenv(gin.EnvGinMode) == "" { // Don't override an explicit env setting.
		gin.SetMode(gin.TestMode)
	}
	r := gin.New()
	return r
}

type testAPI struct {
	*API
	Router *gin.Engine
}

func newTestAPI(t *testing.T, e *engine.Engine) *testAPI {
	r := ginEngine()
	a, err := New(e, nil, r)
	require.NoError(t, err)
	return &testAPI{API: a, Router: r}
}

func do(t *testing.T, a *testAPI, method, url string, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var r io.Reader
	if body != nil {
		j, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		w.Code = http.StatusBadRequest
		fmt.Fprintln(w, err.Error())
	} else {
		a.Router.ServeHTTP(w, req)
	}
	return w
}

func assertDo[T any](t *testing.T, a *testAPI, method, url string, req any, code int, want T) {
	t.Helper()
	w := do(t, a, method, url, req)
	if !assert.Equal(t, code, w.Code, w.Body.String()) {
		return
	}
	var got T
	if !assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got), "body: %v", w.Body.String()) {
		return
	}
	Normalize(want)
	Normalize(got)
	assert.Equal(t, want, got, "request: %+v", req)

}

func testEngine(t *testing.T) (e *engine.Engine) {
	t.Helper()
	d := mock.Domain("mock")
	a, b := d.Class("a"), d.Class("b")
	s := mock.NewStore(d)
	s.AddQuery("mock:a:x", "ax")
	s.AddQuery("mock:b:y", "by")
	r := mock.NewRule("a-b", list(a), list(b), mock.NewQuery(b, "y"))
	e, err := engine.Build().Domains(d).Stores(s).Rules(r).Engine()
	require.NoError(t, err)
	return e
}

func list[T any](x ...T) []T { return x }
