// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_GetDomains(t *testing.T) {
	e, err := engine.Build().
		Domains(mock.NewDomain("foo"), mock.NewDomain("bar")).
		StoreConfigs(
			config.Store{"domain": "foo", "a": "1"},
			config.Store{"domain": "foo", "b": "2"},
			config.Store{"domain": "bar", "x": "y"},
		).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	assertDo(t, a, "GET", "/api/v1alpha1/domains", nil, http.StatusOK, []Domain{
		{Name: "bar", Description: "Mock domain.", Stores: []Store{{"domain": "bar", "x": "y"}}},
		{Name: "foo", Description: "Mock domain.", Stores: []Store{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}}},
	})
}

func TestAPI_ListDomainClasses(t *testing.T) {
	e, err := engine.Build().Domains(mock.NewDomain("foo", "a", "b")).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	// Test with valid domain
	assertDo(t, a, "GET", "/api/v1alpha1/domain/foo/classes", nil, http.StatusOK, []string{"a", "b"})

	// Test with invalid domain
	assertDo(t, a, "GET", "/api/v1alpha1/domain/nonexistent/classes", nil, http.StatusNotFound, Error{Error: "domain not found: nonexistent: unknown domain: nonexistent"})
}

func TestAPIListGoals(t *testing.T) {
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
				Count:   ptr.To(1),
				Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
			},
		})
}

func TestAPIGraphGoals_rules(t *testing.T) {
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
					Count: ptr.To(1),
				},
				{
					Class:   "mock:b",
					Count:   ptr.To(1),
					Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
				}},
			Edges: []Edge{{
				Start: "mock:a",
				Goal:  "mock:b",
				Rules: []Rule{{
					Name:    "a-b",
					Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
				}},
			}},
		})
}

func TestAPIGetObjects(t *testing.T) {
	d := mock.NewDomain("x")
	s := mock.NewStore(d)
	e, err := engine.Build().Domains(d).Stores(s).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	s.AddQuery("x:y:getfoo", "foo")
	w := a.do(t, "GET", "/api/v1alpha1/objects?query=x:y:getfoo", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, `["foo"]`, w.Body.String())

	w = a.do(t, "GET", "/api/v1alpha1/objects?query=x:y:nothing", nil)
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
		if s, ok := body.(string); ok {
			r = strings.NewReader(s)
		} else {
			j, err := json.Marshal(body)
			require.NoError(t, err)
			r = bytes.NewReader(j)
		}
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
	d := mock.NewDomain("mock", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	s := mock.NewStore(d)
	s.AddQuery("mock:a:x", "ax")
	s.AddQuery("mock:b:y", "by")
	s.AddQuery("mock:b:none", nil)
	e, err := engine.Build().Domains(d).Stores(s).Rules(
		mock.NewRule("a-b", list(a), list(b), mock.NewQuery(b, "y")),
		mock.NewRule("a-none", list(a), list(b), mock.NewQuery(b, "none")),
	).Engine()
	require.NoError(t, err)
	return e
}

func list[T any](x ...T) []T { return x }

func TestAPIGraphNeighbors(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbors",
		Neighbors{
			Start: Start{Queries: []string{"mock:a:x"}},
			Depth: 5,
		},
		http.StatusOK,
		Graph{
			Nodes: []Node{
				{Class: "mock:a", Count: ptr.To(1), Queries: []QueryCount{{Query: "mock:a:x", Count: ptr.To(1)}}},
				{Class: "mock:b", Count: ptr.To(1), Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}}},
			},
			Edges: []Edge{{Start: "mock:a", Goal: "mock:b"}},
		})
}

func TestAPIGraphNeighbors_badRequest(t *testing.T) {
	a := newTestAPI(t, testEngine(t))
	w := a.do(t, "POST", "/api/v1alpha1/graphs/neighbors", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConsoleState(t *testing.T) {
	cs := NewConsoleState()
	// Initial state is empty
	assert.Empty(t, cs.Get().View)

	// Set and get
	view := "test"
	cs.Set(&Console{View: view})
	got := cs.Get()
	require.NotNil(t, got.View)
	assert.Equal(t, "test", got.View)

	// Get returns a deep copy - mutating it doesn't affect state
	got.View = "mutated"
	assert.Equal(t, "test", cs.Get().View)
}

func TestConsoleUpdatesSend(t *testing.T) {
	cs := NewConsoleState()

	// Send with no receiver times out
	view := "test"
	err := cs.Send(&Console{View: view})
	assert.Error(t, err)

	// Send with receiver succeeds
	go func() { <-cs.Updates }()
	err = cs.Send(&Console{View: view})
	assert.NoError(t, err)
}

func TestDeepCopy(t *testing.T) {
	view := "hello"
	src := Console{View: view}
	var dst Console
	require.NoError(t, DeepCopy(&dst, src))
	require.NotNil(t, dst.View)
	assert.Equal(t, "hello", dst.View)
	// Verify it's a deep copy
	dst.View = "modified"
	assert.Equal(t, "hello", src.View)
}

func TestTraverseStart_errors(t *testing.T) {
	e := testEngine(t)
	// No class or queries
	_, err := TraverseStart(e, Start{})
	assert.ErrorContains(t, err, "no class")

	// Invalid class
	_, err = TraverseStart(e, Start{Class: "bad:class"})
	assert.Error(t, err)

	// Mismatched query classes
	_, err = TraverseStart(e, Start{Queries: []string{"mock:a:x", "mock:b:y"}})
	assert.ErrorContains(t, err, "mismatch")
}
