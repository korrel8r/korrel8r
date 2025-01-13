// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bytes"
	"encoding/json"
	"errors"
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
	assertDo(t, a, "GET", "/api/v1alpha1/domains", nil, http.StatusOK, []Domain{
		{Name: "bar", Stores: []config.Store{{"domain": "bar", "x": "y"}}},
		{Name: "foo", Stores: []config.Store{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}}},
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
		http.StatusOK, []Node{
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
		http.StatusOK,
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
		http.StatusOK,
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

func TestAPI_PostNeighbours_partial(t *testing.T) {
	e := testEngine(t)
	s := e.StoresFor(e.Domains()[0])[0].(*mock.Store)
	s.AddQuery("mock:b:y", errors.New("oh dear"))
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbours{
			Start: Start{Queries: []string{"mock:a:x"}},
			Depth: 1,
		},
		http.StatusPartialContent,
		Graph{
			Nodes: []Node{{
				Class:   "mock:a",
				Queries: []QueryCount{QueryCount{Query: "mock:a:x", Count: 1}},
				Count:   1,
			},
			},
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
		http.StatusOK, Graph{},
	)
}

func TestAPI_PostNeighbours_none(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbours{
			Start: Start{Queries: []string{"mock:b:y"}},
			Depth: 0,
		},
		http.StatusOK,
		Graph{
			Nodes: []Node{{
				Class:   "mock:b",
				Queries: []QueryCount{{Query: "mock:b:y", Count: 1}},
				Count:   1,
			}}},
	)
}

func TestAPI_GetObjects_empty(t *testing.T) {
	d := mock.Domain("x")
	c := d.Class("y")
	q := mock.NewQuery(c, "test")
	s := mock.NewStore(d)
	e, err := engine.Build().Domains(d).Stores(s).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := a.do(t, "GET", "/api/v1alpha1/objects?query="+url.QueryEscape(q.String()), nil)
	require.Equal(t, http.StatusOK, w.Code)
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

func (a *testAPI) do(t *testing.T, method, url string, body any) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	var r io.Reader
	if body != nil {
		j, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(j)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		rr.Code = http.StatusBadRequest
		fmt.Fprintln(rr, err.Error())
	} else {
		a.Router.ServeHTTP(rr, req)
	}
	return rr
}

func assertDo[T any](t *testing.T, a *testAPI, method, url string, req any, code int, want T) {
	t.Helper()
	rr := a.do(t, method, url, req)
	if !assert.Equal(t, code, rr.Code, rr.Body.String()) {
		return
	}
	var got T
	if !assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got), "body: %v", rr.Body.String()) {
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
