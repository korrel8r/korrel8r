// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeName(t *testing.T) {
	assert.Equal(t, "int", TypeName(0))
	assert.Equal(t, "string", TypeName(""))
}

func TestTypeAssert(t *testing.T) {
	v, err := TypeAssert[string](any("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", v)

	_, err = TypeAssert[string](any(42))
	assert.ErrorContains(t, err, "wrong type")
}

func TestUnmarshal(t *testing.T) {
	var v struct{ Name string }
	require.NoError(t, Unmarshal([]byte(`{"name":"test"}`), &v))
	assert.Equal(t, "test", v.Name)

	v = struct{ Name string }{}
	require.NoError(t, Unmarshal([]byte("name: test"), &v))
	assert.Equal(t, "test", v.Name)
}

func TestUnmarshalAs(t *testing.T) {
	type T struct{ X int }
	v, err := UnmarshalAs[T]([]byte(`{"x": 42}`))
	require.NoError(t, err)
	assert.Equal(t, 42, v.X)

	_, err = UnmarshalAs[T]([]byte(`not valid`))
	assert.Error(t, err)
}

func TestPreview(t *testing.T) {
	f := func(s string) string { return "got:" + s }
	assert.Equal(t, "got:hello", Preview("hello", f))
	assert.Contains(t, Preview(42, f), "42")
}

func TestGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
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
	})

	t.Run("HTTP error with body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL)
		var got map[string]string
		err := Get(context.Background(), u, srv.Client(), &got)
		assert.ErrorContains(t, err, "404")
	})

	t.Run("HTTP error empty body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL)
		var got map[string]string
		err := Get(context.Background(), u, srv.Client(), &got)
		assert.ErrorContains(t, err, "500")
	})
}
