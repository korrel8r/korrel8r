// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeName(t *testing.T) {
	assert.Equal(t, "int", TypeName(0))
	assert.Equal(t, "string", TypeName(""))
}

func TestTypeAssert_Success(t *testing.T) {
	var x any = "hello"
	v, err := TypeAssert[string](x)
	require.NoError(t, err)
	assert.Equal(t, "hello", v)
}

func TestTypeAssert_Failure(t *testing.T) {
	var x any = 42
	_, err := TypeAssert[string](x)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrong type")
}

func TestUnmarshal(t *testing.T) {
	var v struct{ Name string }
	require.NoError(t, Unmarshal([]byte(`{"name":"test"}`), &v))
	assert.Equal(t, "test", v.Name)
}

func TestUnmarshal_YAML(t *testing.T) {
	var v struct{ Name string }
	require.NoError(t, Unmarshal([]byte("name: test"), &v))
	assert.Equal(t, "test", v.Name)
}

func TestUnmarshalAs(t *testing.T) {
	type T struct{ X int }
	v, err := UnmarshalAs[T]([]byte(`{"x": 42}`))
	require.NoError(t, err)
	assert.Equal(t, 42, v.X)
}

func TestUnmarshalAs_Error(t *testing.T) {
	type T struct{ X int }
	_, err := UnmarshalAs[T]([]byte(`not valid`))
	assert.Error(t, err)
}

func TestPreview_MatchingType(t *testing.T) {
	f := func(s string) string { return "got:" + s }
	assert.Equal(t, "got:hello", Preview("hello", f))
}

func TestPreview_WrongType(t *testing.T) {
	f := func(s string) string { return "got:" + s }
	assert.Contains(t, Preview(42, f), "42")
}

func TestDomain_String(t *testing.T) {
	d := NewDomain("testdomain", "A test domain")
	assert.Equal(t, "testdomain", d.String())
}

func TestDomain_ClassMethods(t *testing.T) {
	c := mock.NewDomain("d", "a", "b")
	d := NewDomain("d", "desc", c.Classes()...)
	assert.Equal(t, "a", d.Class("a").Name())
	assert.Nil(t, d.Class("nonexistent"))
	assert.Len(t, d.Classes(), 2)
}

func TestStore_Domain(t *testing.T) {
	d := mock.NewDomain("test")
	s := NewStore(d)
	assert.Equal(t, d, s.Domain())
}

func TestGet_Success(t *testing.T) {
	want := map[string]string{"key": "value"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(want))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	var got map[string]string
	err := Get(context.Background(), u, srv.Client(), &got)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	var got map[string]string
	err := Get(context.Background(), u, srv.Client(), &got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestGet_EmptyErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	var got map[string]string
	err := Get(context.Background(), u, srv.Client(), &got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestTryStores_FirstSucceeds(t *testing.T) {
	d := mock.NewDomain("test", "c")
	s1 := mock.NewStore(d)
	q := mock.NewQuery(d.Class("c"), "selector", "result1")
	s1.AddQuery(q, []korrel8r.Object{"result1"})

	s2 := mock.NewStore(d)
	s2.AddQuery(q, []korrel8r.Object{"result2"})

	ts := TryStores{s1, s2}
	assert.Equal(t, d, ts.Domain())

	var r mock.Result
	err := ts.Get(context.Background(), q, nil, &r)
	require.NoError(t, err)
	assert.Contains(t, r.List(), korrel8r.Object("result1"))
}

func TestTryStores_FallbackOnError(t *testing.T) {
	d := mock.NewDomain("test", "c")
	q := mock.NewQuery(d.Class("c"), "selector")

	s1 := mock.NewStore(d)
	s1.AddQuery(q, fmt.Errorf("store1 error"))

	s2 := mock.NewStore(d)
	s2.AddQuery(q, []korrel8r.Object{"result2"})

	ts := TryStores{s1, s2}

	var r mock.Result
	err := ts.Get(context.Background(), q, nil, &r)
	require.NoError(t, err)
	assert.Contains(t, r.List(), korrel8r.Object("result2"))
}

func TestTryStores_AllFail(t *testing.T) {
	d := mock.NewDomain("test", "c")
	q := mock.NewQuery(d.Class("c"), "selector")

	s1 := mock.NewStore(d)
	s1.AddQuery(q, fmt.Errorf("error1"))

	s2 := mock.NewStore(d)
	s2.AddQuery(q, fmt.Errorf("error2"))

	ts := TryStores{s1, s2}

	var r mock.Result
	err := ts.Get(context.Background(), q, nil, &r)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error1")
	assert.Contains(t, err.Error(), "error2")
}
