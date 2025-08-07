// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"errors"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryStores_Get_FirstStoreSucceeds(t *testing.T) {
	ctx := context.Background()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)

	// Setup successful query for store1
	query := mock.NewQuery(class, "query1")
	store1.AddQuery(query, []korrel8r.Object{"result1", "result2"})

	// Setup query that would return different results for store2 (should not be called)
	store2.AddQuery(query, []korrel8r.Object{"different_result"})

	tryStores := TryStores{store1, store2}

	result := &mock.Result{}
	err := tryStores.Get(ctx, query, nil, result)

	require.NoError(t, err)
	assert.Equal(t, []korrel8r.Object{"result1", "result2"}, result.List())
}

func TestTryStores_Get_SecondStoreSucceeds(t *testing.T) {
	ctx := context.Background()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)

	query := mock.NewQuery(class, "query1")

	// Setup store1 to fail
	store1Error := errors.New("store1 failed")
	store1.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, store1Error
		}
		return nil, nil
	})

	// Setup store2 to succeed
	store2.AddQuery(query, []korrel8r.Object{"result_from_store2"})

	tryStores := TryStores{store1, store2}

	result := &mock.Result{}
	err := tryStores.Get(ctx, query, nil, result)

	require.NoError(t, err)
	assert.Equal(t, []korrel8r.Object{"result_from_store2"}, result.List())
}

func TestTryStores_Get_AllStoresFail(t *testing.T) {
	ctx := context.Background()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)

	query := mock.NewQuery(class, "query1")

	// Setup both stores to fail
	store1Error := errors.New("store1 error")
	store2Error := errors.New("store2 error")

	store1.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, store1Error
		}
		return nil, nil
	})

	store2.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, store2Error
		}
		return nil, nil
	})

	tryStores := TryStores{store1, store2}

	result := &mock.Result{}
	err := tryStores.Get(ctx, query, nil, result)

	require.Error(t, err)
	// The error should contain both store errors (joined)
	assert.Contains(t, err.Error(), "store1 error")
	assert.Contains(t, err.Error(), "store2 error")
	assert.Empty(t, result.List())
}

func TestTryStores_Get_EmptyStores(t *testing.T) {
	ctx := context.Background()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")
	query := mock.NewQuery(class, "query1")

	tryStores := TryStores{}

	result := &mock.Result{}
	err := tryStores.Get(ctx, query, nil, result)

	require.NoError(t, err) // No stores means no errors
	assert.Empty(t, result.List())
}

func TestTryStores_Get_WithErrors_Mixed(t *testing.T) {
	ctx := context.Background()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)
	store3 := mock.NewStore(domain, class)

	query := mock.NewQuery(class, "query1")

	// store1 fails
	store1Error := errors.New("store1 failed")
	store1.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, store1Error
		}
		return nil, nil
	})

	// store2 succeeds - this should stop the iteration
	store2.AddQuery(query, []korrel8r.Object{"success_result"})

	// store3 would fail if called, but shouldn't be called
	store3Error := errors.New("store3 should not be called")
	store3.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		return nil, store3Error
	})

	tryStores := TryStores{store1, store2, store3}

	result := &mock.Result{}
	err := tryStores.Get(ctx, query, nil, result)

	require.NoError(t, err)
	assert.Equal(t, []korrel8r.Object{"success_result"}, result.List())
}

func TestTryStores_Get_QueryNotFound(t *testing.T) {
	ctx := context.Background()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)

	query := mock.NewQuery(class, "nonexistent_query")

	// Both stores don't have this query (default behavior returns no results, no error)

	tryStores := TryStores{store1, store2}

	result := &mock.Result{}
	err := tryStores.Get(ctx, query, nil, result)

	require.NoError(t, err)
	assert.Empty(t, result.List())
}
