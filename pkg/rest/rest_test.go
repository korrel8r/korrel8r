// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
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
		{Name: "bar", Stores: []Store{{"domain": "bar", "x": "y"}}},
		{Name: "foo", Stores: []Store{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}}},
	})
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

func TestAPIGraphGoals_bad_goal(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/goals?zeros=true",
		Goals{
			Start: Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Goals: []string{"mock:nosuchclass"},
		},
		http.StatusBadRequest,
		Graph{})
}

func TestAPIPostNeighbors(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbors",
		Neighbors{
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
					Count: ptr.To(1),
				},
				{
					Class:   "mock:b",
					Count:   ptr.To(1),
					Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
				}},
			Edges: []Edge{{Start: "mock:a", Goal: "mock:b"}},
		},
	)
}

// Alternate spelling
func TestAPIPostNeighbours(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbours",
		Neighbors{
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
					Count: ptr.To(1),
				},
				{
					Class:   "mock:b",
					Count:   ptr.To(1),
					Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
				}},
			Edges: []Edge{{Start: "mock:a", Goal: "mock:b"}},
		},
	)
}

func TestAPIPostNeighborsError(t *testing.T) {
	e := testEngine(t)
	s := e.StoresFor(e.Domains()[0])[0].(*mock.Store)
	s.AddQuery("mock:b:y", errors.New("oh dear"))
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbors",
		Neighbors{
			Start: Start{Queries: []string{"mock:a:x"}},
			Depth: 1,
		},
		http.StatusOK,
		Graph{
			Nodes: []Node{{
				Class:   "mock:a",
				Queries: []QueryCount{{Query: "mock:a:x", Count: ptr.To(1)}},
				Count:   ptr.To(1),
			},
			},
		},
	)
}

func TestAPIPostNeighborsEmpty(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbors",
		Neighbors{
			Start: Start{Queries: []string{"mock:a:nossuchthing"}},
			Depth: 1,
		},
		http.StatusOK, Graph{},
	)
}

func TestAPIPostNeighborsNone(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbors",
		Neighbors{
			Start: Start{Queries: []string{"mock:b:y"}},
			Depth: 0,
		},
		http.StatusOK,
		Graph{
			Nodes: []Node{{
				Class:   "mock:b",
				Queries: []QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
				Count:   ptr.To(1),
			}}},
	)
}

func TestAPIPostNeighborsInvalidClass(t *testing.T) {
	e := testEngine(t)
	a := newTestAPI(t, e)
	w := a.do(t, "POST", "/api/v1alpha1/graphs/neighbors",
		Neighbors{
			Start: Start{Queries: []string{"mock:b:y"}, Class: "not-a-class"},
			Depth: 0,
		})
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, `{"error":"invalid class name: not-a-class"}`, w.Body.String())
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

// blockingStore Get blocks until the context is cancelled.
type blockingStore struct{ domain korrel8r.Domain }

func (s *blockingStore) Domain() korrel8r.Domain { return s.domain }

func (s *blockingStore) Get(ctx context.Context, _ korrel8r.Query, _ *korrel8r.Constraint, _ korrel8r.Appender) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestConfigTuningTimeout(t *testing.T) {
	d := mock.NewDomain("x")
	s := &blockingStore{d}
	timeout := 10 * time.Millisecond
	cfg := config.Config{
		Tuning: &config.Tuning{RequestTimeout: ptr.To(config.Duration{Duration: timeout})},
	}
	e, err := engine.Build().Domains(d).Stores(s).Config(config.Configs{cfg}).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	start := time.Now()
	w := a.do(t, "GET", "/api/v1alpha1/objects?query=x:y:z", nil)
	delay := time.Since(start)
	assert.InEpsilon(t, timeout, delay, 0.1, "delay: %v", delay)
	// TODO this should be StatusRequestTimeout
	assert.Equal(t, http.StatusNotFound, w.Code)
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
