// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	logDomain "github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
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
	a, x, y, z := apiRules(t)
	assertDo(t, a, "POST", "/api/v1alpha1/lists/goals",
		Goals{
			Start: Start{
				Class:   x.String(),
				Queries: []string{x.String() + `:["a"]`},
				Objects: []json.RawMessage{[]byte(`"b"`)},
			},
			Goals: []string{y.String(), z.String()},
		},
		200, []Node{
			{
				Class: "bar:y",
				Count: 2,
				Queries: []QueryCount{
					{Query: mock.NewQuery(y, "aa").String(), Count: 1},
					{Query: mock.NewQuery(y, "bb").String(), Count: 1},
				},
			},
			{
				Class: "bar:z",
				Count: 1,
				Queries: []QueryCount{
					{Query: mock.NewQuery(z, "c").String(), Count: 1},
				},
			},
		})
}

func TestAPI_GraphGoals_rules(t *testing.T) {
	a, x, y, z := apiRules(t)
	yQueries := []QueryCount{
		{Query: mock.NewQuery(y, "aa").String(), Count: 1},
		{Query: mock.NewQuery(y, "bb").String(), Count: 1},
	}
	zQueries := []QueryCount{
		{Query: mock.NewQuery(z, "c").String(), Count: 1},
	}
	xQuery := mock.NewQuery(x, "a").String()
	assertDo(t, a, "POST", "/api/v1alpha1/graphs/goals?rules=true",
		Goals{
			Start: Start{
				Class:   x.String(),
				Queries: []string{xQuery},
				Objects: []json.RawMessage{[]byte(`"b"`)},
			},
			Goals: []string{z.String()},
		},
		200,
		Graph{
			Nodes: []Node{
				{Class: "foo:x", Count: 2, Queries: []QueryCount{{Query: xQuery, Count: 1}}},
				{Class: "bar:y", Count: 2, Queries: yQueries},
				{Class: "bar:z", Count: 1, Queries: zQueries},
			},
			Edges: []Edge{
				{Start: "foo:x", Goal: "bar:y", Rules: []Rule{{Name: "x-y", Queries: yQueries}}},
				{Start: "bar:y", Goal: "bar:z", Rules: []Rule{{Name: "y-z", Queries: zQueries}}},
			},
		})
}

func TestAPI_PostNeighbours_noRules(t *testing.T) {
	a, x, y, _ := apiRules(t)
	qc := []QueryCount{{Query: mock.NewQuery(y, "aa").String(), Count: 1}}
	assertDo(t, a, "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbours{
			Start: Start{
				Class:   x.String(),
				Objects: []json.RawMessage{[]byte(`"a"`)},
			},
			Depth: 1,
		},
		200,
		Graph{
			Nodes: []Node{
				{Class: "foo:x", Count: 1},
				{Class: "bar:y", Count: 1, Queries: qc},
			},
			Edges: []Edge{
				{Start: "foo:x", Goal: "bar:y"},
			},
		},
	)
}

func TestAPI_GetObjects(t *testing.T) {
	want := []any{"a", "b", "c"}
	d := mock.Domain("x")
	c := d.Class("y")
	q := mock.NewQuery(c, want...)
	s, err := mock.NewStore(d, nil)
	require.NoError(t, err)
	e, err := engine.Build().Domains(d).Stores(s).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	assertDo(t, a, "GET", "/api/v1alpha1/objects?query="+url.QueryEscape(q.String()), nil, 200, want)
}

func TestAPI_GetObjects_empty(t *testing.T) {
	d := mock.Domain("x")
	c := d.Class("y")
	q := mock.NewQuery(c)
	s, err := mock.NewStore(d, nil)
	require.NoError(t, err)
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
		r = strings.NewReader(test.JSONString(body))
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

// normalize values by sorting slices to avoid test failure due to ordering inconsistency.
func normalize(v any) {
	switch v := v.(type) {
	case Graph:
		normalize(v.Nodes)
		normalize(v.Edges)
	case []Node:
		slices.SortFunc(v, func(a, b Node) int { return strings.Compare(a.Class, b.Class) })
		for _, n := range v {
			normalize(n)
		}
	case []Edge:
		slices.SortFunc(v, func(a, b Edge) int {
			if n := strings.Compare(a.Start, b.Start); n != 0 {
				return n
			} else {
				return strings.Compare(a.Goal, b.Goal)
			}
		})
		for _, e := range v {
			normalize(e)
		}
	case Node:
		normalize(v.Queries)
	case Edge:
		for _, r := range v.Rules {
			normalize(r.Queries)
		}
	case []QueryCount:
		slices.SortFunc(v, func(a, b QueryCount) int { return strings.Compare(a.Query, b.Query) })
	}
}

func assertDo[T any](t *testing.T, a *testAPI, method, url string, req any, code int, want T) {
	t.Helper()
	w := do(t, a, method, url, req)
	if assert.Equal(t, code, w.Code, w.Body.String()) {
		var got T
		if assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got), "body: %v", w.Body.String()) {
			normalize(want)
			normalize(got)
			if assert.JSONEq(t, test.JSONPretty(want), test.JSONPretty(got)) {
				return
			}
		}
	}
	t.Logf("request: %v", test.JSONString(req)) // Log the request body on error.
}

// doubleFunc returns a goal object with the name of the start object repeated twice.
func doubleFunc(goal korrel8r.Class) func(korrel8r.Object) (korrel8r.Query, error) {
	return func(o korrel8r.Object) (korrel8r.Query, error) {
		return mock.NewQuery(goal, o.(string)+o.(string)), nil
	}
}

func apiRules(t *testing.T) (a *testAPI, x, y, z korrel8r.Class) {
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	x, y, z = foo.Class("x"), bar.Class("y"), bar.Class("z")
	var stores []korrel8r.Store
	for _, d := range []korrel8r.Domain{foo, bar} {
		s, err := mock.NewStore(d, nil)
		require.NoError(t, err)
		stores = append(stores, s)
	}
	e, err := engine.Build().
		Domains(foo, bar).
		Stores(stores...).
		Rules(mock.NewApplyRule("x-y", x, y, doubleFunc(y)), mock.NewQueryRule("y-z", y, mock.NewQuery(z, "c"))).
		Engine()
	require.NoError(t, err)
	api := newTestAPI(t, e)
	return api, x, y, z
}
