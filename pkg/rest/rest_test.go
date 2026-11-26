// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/session"
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
	assertDo(t, a, "GET", "/api/v1alpha1/domains", nil, http.StatusOK, []api.Domain{
		{Name: "bar", Description: "Mock domain.", Stores: []api.Store{{"domain": "bar", "x": "y"}}},
		{Name: "foo", Description: "Mock domain.", Stores: []api.Store{{"domain": "foo", "a": "1"}, {"domain": "foo", "b": "2"}}},
	})
}

func TestAPI_ListDomainClasses(t *testing.T) {
	e, err := engine.Build().Domains(mock.NewDomain("foo", "a", "b")).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	// Test with valid domain
	assertDo(t, a, "GET", "/api/v1alpha1/domain/foo/classes", nil, http.StatusOK, []string{"a", "b"})

	// Test with invalid domain
	assertDo(t, a, "GET", "/api/v1alpha1/domain/nonexistent/classes", nil, http.StatusNotFound, api.Error{Error: "domain not found: nonexistent"})
}

func TestAPIListGoals(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/lists/goals",
		api.Goals{
			Start: api.Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Goals: []string{"mock:b"},
		},
		http.StatusOK, []api.Node{
			{
				Class:   "mock:b",
				Count:   ptr.To(1),
				Queries: []api.QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
			},
		})
}

func TestAPIGraphGoals_rules(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/goals?rules=true",
		api.Goals{
			Start: api.Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Goals: []string{"mock:b"},
		},
		http.StatusOK,
		api.Graph{
			Nodes: []api.Node{
				{
					Class: "mock:a",
					Count: ptr.To(1),
				},
				{
					Class:   "mock:b",
					Count:   ptr.To(1),
					Queries: []api.QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
				}},
			Edges: []api.Edge{{
				Start: "mock:a",
				Goal:  "mock:b",
				Rules: []api.Rule{{
					Name:    "a-b",
					Queries: []api.QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}},
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

func TestAPIGetObjects_withConstraint(t *testing.T) {
	d := mock.NewDomain("x")
	s := mock.NewStore(d)
	s.AddQuery("x:y:many", []korrel8r.Object{"a1", "a2", "a3"})
	e, err := engine.Build().Domains(d).Stores(s).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	// Without constraint: all 3 objects
	w := a.do(t, "GET", "/api/v1alpha1/objects?query=x:y:many", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, `["a1","a2","a3"]`, w.Body.String())

	// With limit=1 constraint: only 1 object
	w = a.do(t, "GET", "/api/v1alpha1/objects?query=x:y:many&limit=1", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, `["a1"]`, w.Body.String())
}

func testMux() *http.ServeMux { return http.NewServeMux() }

type testAPI struct {
	*API
	Router *http.ServeMux
}

func newTestAPI(t *testing.T, e *engine.Engine) *testAPI {
	r := testMux()
	a, err := New(session.NewSingleManager(e), r)
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

func TestAPIGraphGoals_labels(t *testing.T) {
	d := mock.NewDomain("mock", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	s := mock.NewStore(d)
	s.AddQuery("mock:a:x", "ax")
	s.AddQuery("mock:b:y", "by")
	e, err := engine.Build().Domains(d).Stores(s).Rules(
		mock.NewRule("a-b", list(a), list(b), mock.NewQuery(b, "y")),
	).Config(config.Configs{
		{
			StatusRules: []config.StatusRule{
				{
					Name:   "label-b",
					Start:  config.ClassSpec{Domain: "mock", Classes: []string{"b"}},
					Status: `my-label`,
				},
			},
		},
	}).Engine()
	require.NoError(t, err)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/goals",
		api.Goals{
			Start: api.Start{
				Class:   "mock:a",
				Objects: []json.RawMessage{[]byte(`"x"`)},
			},
			Goals: []string{"mock:b"},
		},
		http.StatusOK,
		api.Graph{
			Nodes: []api.Node{
				{
					Class: "mock:a",
					Count: ptr.To(1),
				},
				{
					Class: "mock:b",
					Count: ptr.To(1),
					Queries: []api.QueryCount{{
						Query:    "mock:b:y",
						Count:    ptr.To(1),
						Statuses: []api.StatusCount{{Status: "my-label", Count: ptr.To(1)}},
					}},
				}},
			Edges: []api.Edge{{Start: "mock:a", Goal: "mock:b"}},
		})
}

func TestAPIGraphNeighbors(t *testing.T) {
	e := testEngine(t)
	assertDo(t, newTestAPI(t, e), "POST", "/api/v1alpha1/graphs/neighbors",
		api.Neighbors{
			Start: api.Start{Queries: []string{"mock:a:x"}},
			Depth: 5,
		},
		http.StatusOK,
		api.Graph{
			Nodes: []api.Node{
				{Class: "mock:a", Count: ptr.To(1), Queries: []api.QueryCount{{Query: "mock:a:x", Count: ptr.To(1)}}},
				{Class: "mock:b", Count: ptr.To(1), Queries: []api.QueryCount{{Query: "mock:b:y", Count: ptr.To(1)}}},
			},
			Edges: []api.Edge{{Start: "mock:a", Goal: "mock:b"}},
		})
}

func TestAPIGraphNeighbors_truncated(t *testing.T) {
	limit := 1
	a := newTestAPI(t, testEngine(t))
	w := a.do(t, "POST", "/api/v1alpha1/graphs/neighbors", api.Neighbors{
		Start: api.Start{
			Queries:    []string{"mock:a:x"},
			Constraint: &api.Constraint{TotalLimit: &limit},
		},
		Depth: 5,
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-Korrel8r-Truncated"))
	assert.Equal(t, "totalLimit", w.Header().Get("X-Korrel8r-Truncated-By"))
	var got api.Graph
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.NotNil(t, got.Truncation)
	assert.Equal(t, api.Truncation{Condition: "totalLimit", Limit: 1}, *got.Truncation)
}

func TestAPIGoals_badRequest(t *testing.T) {
	a := newTestAPI(t, testEngine(t))
	for _, path := range []string{"/api/v1alpha1/graphs/goals", "/api/v1alpha1/lists/goals"} {
		t.Run(path, func(t *testing.T) {
			w := a.do(t, "POST", path, `not json`)
			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.JSONEq(t, `{"error":"invalid character 'o' in literal null (expecting 'u')"}`, w.Body.String())
		})
	}
}

func TestAPIGraphNeighbors_badRequest(t *testing.T) {
	a := newTestAPI(t, testEngine(t))
	w := a.do(t, "POST", "/api/v1alpha1/graphs/neighbors", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"invalid character 'o' in literal null (expecting 'u')"}`, w.Body.String())
}

func TestHTTPRoutesAndValidation(t *testing.T) {
	a := newTestAPI(t, testEngine(t))
	for _, tt := range []struct {
		method, path string
		status       int
		allow        string
	}{
		{"GET", "/api/v1alpha1/domain/mock/classes", http.StatusOK, ""},
		{"GET", "/api/v1alpha1/does-not-exist", http.StatusNotFound, ""},
		{"POST", "/api/v1alpha1/domains", http.StatusMethodNotAllowed, "GET, HEAD"},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := a.do(t, tt.method, tt.path, nil)
			assert.Equal(t, tt.status, w.Code, w.Body.String())
			if tt.allow != "" {
				assert.Equal(t, tt.allow, w.Header().Get("Allow"))
			}
		})
	}

	w := a.do(t, "PUT", "/api/v1alpha1/config?verbose=not-an-integer", nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	var got api.Error
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Contains(t, got.Error, "Invalid format for parameter verbose")
}

func TestHomePageListsAllPaths(t *testing.T) {
	a := newTestAPI(t, testEngine(t))
	a.AddPaths("/metrics", "/debug/pprof/")
	w := a.do(t, "GET", "/", nil)
	require.Equal(t, http.StatusOK, w.Code)
	lines := strings.Split(w.Body.String(), "\n")
	for _, path := range []string{"/", api.BasePath, api.BasePath + "/domains", "/metrics", "/debug/pprof/"} {
		assert.Contains(t, lines, path)
	}
}

func TestRecoverFromPanic(t *testing.T) {
	t.Run("before response", func(t *testing.T) {
		h := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
		w := httptest.NewRecorder()
		assert.NotPanics(t, func() { h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil)) })
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, `{"error":"internal error"}`, w.Body.String()) // No panic details leaked.
	})

	t.Run("after response", func(t *testing.T) {
		h := Recover(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []string{"partial"})
			panic("boom")
		}))
		w := httptest.NewRecorder()
		assert.NotPanics(t, func() { h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil)) })
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, `["partial"]`, w.Body.String())
	})

	t.Run("abort is not recovered", func(t *testing.T) {
		h := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
		assert.PanicsWithError(t, http.ErrAbortHandler.Error(), func() {
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
		})
	})
}

func TestSharedResponseWriter(t *testing.T) {
	rw := newResponseWriter(httptest.NewRecorder())
	assert.Same(t, rw, newResponseWriter(rw), "chained middleware must share one wrapper")
}

func TestResponseWriterDoesNotBufferStreams(t *testing.T) {
	w := newResponseWriter(httptest.NewRecorder())
	for range 100 {
		_, err := w.Write([]byte("event: update\n\n"))
		require.NoError(t, err)
		w.Flush()
	}
	assert.Nil(t, w.body, "normal requests and SSE streams must not accumulate a response body")

	w.body = &bytes.Buffer{} // Detailed logging also must not buffer an SSE stream.
	w.Header().Set("Content-Type", "text/event-stream")
	_, err := w.Write([]byte("event: update\n\n"))
	require.NoError(t, err)
	assert.Zero(t, w.body.Len())
}

func TestMetricsUseRoutePattern(t *testing.T) {
	a := newTestAPI(t, testEngine(t))
	path := "/api/v1alpha1/domain/untrusted-domain/classes"
	w := a.do(t, http.MethodGet, path, nil)
	require.Equal(t, http.StatusNotFound, w.Code)

	cachedRequestAttrs.mu.Lock()
	defer cachedRequestAttrs.mu.Unlock()
	_, ok := cachedRequestAttrs.attrs[requestMetricKey{"GET", "/api/v1alpha1/domain/{domain}/classes", "404"}]
	assert.True(t, ok, "metrics must use the registered pattern")
	for key := range cachedRequestAttrs.attrs {
		assert.NotContains(t, key.path, "untrusted-domain", "metric label must not contain URL parameters")
	}
}

func TestHTTPRoutingAndAuth(t *testing.T) {
	sessions := session.NewTokenReviewManager(test.FakeTokenReview(), time.Hour, func() (*engine.Engine, error) {
		return testEngine(t), nil
	})
	mux := http.NewServeMux()
	_, err := New(sessions, mux)
	require.NoError(t, err)
	assert.Contains(t, WebProfile(mux, sessions), "/debug/pprof/")
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/mcp", session.Middleware(sessions)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	for _, path := range []string{"/api/v1alpha1/domains", "/api/v1alpha1", "/debug/pprof/", "/mcp"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code, path)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"error":"session authentication error: no bearer token in request"}`, w.Body.String())
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, w.Code, "metrics must remain public")

	for _, tt := range []struct {
		path   string
		status int
	}{
		{"/api/v1alpha1/domains", http.StatusOK},
		{"/debug/pprof/", http.StatusOK},
		{"/mcp", http.StatusNoContent},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		req.Header.Set("Authorization", "Bearer token-A")
		mux.ServeHTTP(w, req)
		assert.Equal(t, tt.status, w.Code, tt.path, w.Body.String())
	}
}

func TestMultiSession_QueryIsolation(t *testing.T) {
	// Each session gets a separate engine with different store data.
	// Verify that REST requests with different auth tokens get different results.
	var callCount atomic.Int32
	factory := func() (*engine.Engine, error) {
		n := callCount.Add(1)
		d := mock.NewDomain("mock", "a")
		s := mock.NewStore(d)
		s.AddQuery("mock:a:q", fmt.Sprintf("result-%d", n))
		return engine.Build().Domains(d).Stores(s).Engine()
	}
	sessions := session.NewTokenReviewManager(test.FakeTokenReview(), time.Hour, factory)

	r := testMux()
	_, err := New(sessions, r)
	require.NoError(t, err)

	getObjects := func(token string) string {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1alpha1/objects?query=mock:a:q", nil)
		req.Header.Set("Authorization", token)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		return w.Body.String()
	}

	resultA := getObjects("Bearer token-A")
	resultB := getObjects("Bearer token-B")
	assert.NotEqual(t, resultA, resultB, "different tokens should get results from different engines")

	// Same token should get the same result (same session reused).
	resultA2 := getObjects("Bearer token-A")
	assert.Equal(t, resultA, resultA2, "same token should reuse session")
}

func TestAPI_Help(t *testing.T) {
	e, err := engine.Build().Domains(mock.NewDomain("mock", "a")).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := a.do(t, "GET", "/api/v1alpha1/help", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var got api.Help
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Contains(t, got.Documentation, "Mock domain.")
}

func TestAPI_HelpDomain(t *testing.T) {
	e, err := engine.Build().Domains(mock.NewDomain("mock", "a"), mock.NewDomain("other")).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := a.do(t, "GET", "/api/v1alpha1/help/mock", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var got api.Help
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Contains(t, got.Documentation, "Mock domain.")
}

func TestAPI_HelpDomain_notFound(t *testing.T) {
	e, err := engine.Build().Domains(mock.NewDomain("mock")).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := a.do(t, "GET", "/api/v1alpha1/help/nosuch", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_GetConsole(t *testing.T) {
	d := mock.NewDomain("mock", "a")
	e, err := engine.Build().Domains(d).Stores(mock.NewStore(d)).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	// Set console state via PUT
	w := a.do(t, "PUT", "/api/v1alpha1/console", api.Console{View: "mock:a:x"})
	require.Equal(t, http.StatusOK, w.Code)

	// Read it back via GET
	w = a.do(t, "GET", "/api/v1alpha1/console", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var got api.Console
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, api.Query("mock:a:x"), got.View)
}

func TestAPI_GetConsole_noConsole(t *testing.T) {
	e, err := engine.Build().Domains(mock.NewDomain("mock")).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := a.do(t, "GET", "/api/v1alpha1/console", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_ShowInConsole(t *testing.T) {
	d := mock.NewDomain("mock", "a")
	e, err := engine.Build().Domains(d).Stores(mock.NewStore(d)).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)

	// Start SSE listener on a real HTTP server
	srv := httptest.NewServer(a.Router)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1alpha1/console/events", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	events := make(chan string, 10)
	go func() {
		defer close(events)
		s := bufio.NewScanner(resp.Body)
		for s.Scan() {
			if data, ok := strings.CutPrefix(s.Text(), "data: "); ok {
				events <- data
			}
		}
	}()

	time.Sleep(100 * time.Millisecond) // Let SSE listener start
	w := a.do(t, "PUT", "/api/v1alpha1/console/events", api.Console{View: "mock:a:x"})
	require.Equal(t, http.StatusOK, w.Code)

	select {
	case data := <-events:
		var got api.Console
		require.NoError(t, json.Unmarshal([]byte(data), &got))
		assert.Equal(t, api.Query("mock:a:x"), got.View)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}

func TestAPI_ShowInConsole_noListener(t *testing.T) {
	d := mock.NewDomain("mock", "a")
	e, err := engine.Build().Domains(d).Stores(mock.NewStore(d)).Engine()
	require.NoError(t, err)
	a := newTestAPI(t, e)
	w := a.do(t, "PUT", "/api/v1alpha1/console/events", api.Console{View: "mock:a:x"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTraverseStart_errors(t *testing.T) {
	e := testEngine(t)
	// No class or queries
	_, err := TraverseStart(e, api.Start{})
	assert.ErrorContains(t, err, "no class")

	// Invalid class
	_, err = TraverseStart(e, api.Start{Class: "bad:class"})
	assert.Error(t, err)

	// Mismatched query classes
	_, err = TraverseStart(e, api.Start{Queries: []string{"mock:a:x", "mock:b:y"}})
	assert.ErrorContains(t, err, "expected class mock:a in query mock:b:y")
}
