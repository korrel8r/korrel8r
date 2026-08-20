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

func tryStoresFixture(t *testing.T) (*mock.Domain, korrel8r.Class, korrel8r.Query) {
	t.Helper()
	domain := mock.NewDomain("test", "testclass")
	class := domain.Class("testclass")
	query := mock.NewQuery(class, "query1")
	return domain, class, query
}

func TestTryStores_Get_FirstStoreSucceeds(t *testing.T) {
	domain, class, query := tryStoresFixture(t)

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)
	store1.AddQuery(query, []korrel8r.Object{"result1", "result2"})
	store2.AddQuery(query, []korrel8r.Object{"different_result"})

	tryStores := TryStores{store1, store2}
	assert.Equal(t, domain, tryStores.Domain())

	result := &mock.Result{}
	err := tryStores.Get(context.Background(), query, nil, result)
	require.NoError(t, err)
	assert.Equal(t, []korrel8r.Object{"result1", "result2"}, result.List())
}

func TestTryStores_Get_FallbackOnError(t *testing.T) {
	domain, class, query := tryStoresFixture(t)

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)

	store1.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, errors.New("store1 failed")
		}
		return nil, nil
	})
	store2.AddQuery(query, []korrel8r.Object{"result_from_store2"})

	result := &mock.Result{}
	err := TryStores{store1, store2}.Get(context.Background(), query, nil, result)
	require.NoError(t, err)
	assert.Equal(t, []korrel8r.Object{"result_from_store2"}, result.List())
}

func TestTryStores_Get_AllStoresFail(t *testing.T) {
	domain, class, query := tryStoresFixture(t)

	store1 := mock.NewStore(domain, class)
	store2 := mock.NewStore(domain, class)

	store1.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, errors.New("store1 error")
		}
		return nil, nil
	})
	store2.AddLookup(func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if q.String() == query.String() {
			return nil, errors.New("store2 error")
		}
		return nil, nil
	})

	result := &mock.Result{}
	err := TryStores{store1, store2}.Get(context.Background(), query, nil, result)
	require.Error(t, err)
	assert.ErrorContains(t, err, "store1 error")
	assert.ErrorContains(t, err, "store2 error")
	assert.Empty(t, result.List())
}

func TestTryStores_Get_EmptyStores(t *testing.T) {
	_, class, _ := tryStoresFixture(t)
	query := mock.NewQuery(class, "query1")

	result := &mock.Result{}
	err := TryStores{}.Get(context.Background(), query, nil, result)
	require.NoError(t, err)
	assert.Empty(t, result.List())
}
