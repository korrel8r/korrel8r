// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package podlog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainQuery(t *testing.T) {
	q, err := Domain.Query("podlog:log:{namespace: netobserv, containers: [opa]}")
	require.NoError(t, err)
	want := &Query{Selector: k8s.Selector{Namespace: "netobserv"}, Containers: []string{"opa"}}
	require.Equal(t, want, q)
}

func TestClassMethods(t *testing.T) {
	c := Class{}

	t.Run("Domain", func(t *testing.T) {
		assert.Equal(t, Domain, c.Domain())
	})

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "log", c.Name())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "podlog:log", c.String())
	})

	t.Run("Unmarshal", func(t *testing.T) {
		testObj := Object{
			Body:      "test log message",
			Timestamp: time.Now(),
			Attributes: map[string]any{
				"k8s.pod.name": "test-pod",
			},
		}
		data, err := json.Marshal(testObj)
		require.NoError(t, err)

		obj, err := c.Unmarshal(data)
		require.NoError(t, err)
		assert.Equal(t, test.JSONString(testObj), test.JSONString(obj))
	})

	t.Run("Preview", func(t *testing.T) {
		testObj := Object{Body: "test log message"}
		preview := c.Preview(testObj)
		assert.Equal(t, "test log message", preview)

		// Test with non-Object type
		preview = c.Preview("not an object")
		assert.Equal(t, "", preview)
	})
}

func TestQueryMethods(t *testing.T) {
	q := &Query{
		Selector:   k8s.Selector{Namespace: "test"},
		Containers: []string{"foobar"},
	}

	t.Run("Class", func(t *testing.T) {
		assert.Equal(t, Class{}, q.Class())
	})

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, `podlog:log:{"namespace":"test","containers":["foobar"]}`, q.String())
	})
}

func TestStoreMethods(t *testing.T) {
	store := &Store{}

	t.Run("Domain", func(t *testing.T) {
		assert.Equal(t, Domain, store.Domain())
	})

	t.Run("StoreClasses", func(t *testing.T) {
		classes, err := store.StoreClasses()
		require.NoError(t, err)
		assert.Equal(t, []korrel8r.Class{Class{}}, classes)
	})
}

func TestDomainQueryErrors(t *testing.T) {
	testCases := []struct {
		name  string
		query string
	}{
		{"invalid format", "invalid"},
		{"wrong domain", "k8s:Pod:{}"},
		{"invalid json", "podlog:log:{invalid json}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Domain.Query(tc.query)
			assert.Error(t, err)
		})
	}
}

func TestNewStoreError(t *testing.T) {
	// Test with nil config should still work (uses default kube config)
	store, err := NewStore(nil, nil)
	if err != nil {
		// This is expected if no kube config is available
		assert.Error(t, err)
		assert.Nil(t, store)
	} else {
		assert.NotNil(t, store)
	}
}

func TestQueryWithContainerField(t *testing.T) {
	t.Run("with container", func(t *testing.T) {
		q, err := Domain.Query("podlog:log:{namespace: test, containers: [app]}")
		require.NoError(t, err)

		query := q.(*Query)
		assert.Equal(t, []string{"app"}, query.Containers)
		assert.Equal(t, "test", query.Namespace)
	})

	t.Run("without container", func(t *testing.T) {
		q, err := Domain.Query("podlog:log:{namespace: test}")
		require.NoError(t, err)

		query := q.(*Query)
		assert.Empty(t, query.Containers)
		assert.Equal(t, "test", query.Namespace)
	})
}

func TestGetInvalidQuery(t *testing.T) {
	store := &Store{}

	// Test with wrong query type
	invalidQuery := k8s.NewQuery(k8s.ClassNamed("Pod"), k8s.Selector{})
	r := result.New(Class{})

	err := store.Get(context.Background(), invalidQuery, nil, r)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "want *podlog.Query")
}
