// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDomains() *korrel8r.Domains {
	ds := korrel8r.NewDomains()
	ds.Add(mock.NewDomain("alpha", "a1", "a2"))
	ds.Add(mock.NewDomain("beta", "b1"))
	return ds
}

func TestDomains_Add_Domain(t *testing.T) {
	ds := newDomains()

	d, err := ds.Domain("alpha")
	require.NoError(t, err)
	assert.Equal(t, "alpha", d.Name())

	_, err = ds.Domain("nosuch")
	assert.Error(t, err)
}

func TestDomains_List(t *testing.T) {
	ds := newDomains()
	list := ds.List()
	require.Len(t, list, 2)
	assert.Equal(t, "alpha", list[0].Name())
	assert.Equal(t, "beta", list[1].Name())
}

func TestDomains_Class(t *testing.T) {
	ds := newDomains()

	c, err := ds.Class("alpha:a1")
	require.NoError(t, err)
	assert.Equal(t, "a1", c.Name())

	_, err = ds.Class("alpha:nosuch")
	assert.Error(t, err)

	_, err = ds.Class("nosuch:a1")
	assert.Error(t, err)

	_, err = ds.Class("invalid")
	assert.Error(t, err)
}

func TestDomains_DomainClass(t *testing.T) {
	ds := newDomains()

	c, err := ds.DomainClass("beta", "b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", c.Name())

	_, err = ds.DomainClass("beta", "nosuch")
	assert.Error(t, err)

	_, err = ds.DomainClass("nosuch", "b1")
	assert.Error(t, err)
}

func TestDomains_Query(t *testing.T) {
	ds := newDomains()

	q, err := ds.Query("alpha:a1:somedata")
	require.NoError(t, err)
	assert.Equal(t, "a1", q.Class().Name())

	_, err = ds.Query("nosuch:x:data")
	assert.Error(t, err)

	_, err = ds.Query("invalid")
	assert.Error(t, err)
}

func TestDomains_QueryCache(t *testing.T) {
	ds := newDomains()
	q1, err := ds.Query("alpha:a1:somedata")
	require.NoError(t, err)
	q2, err := ds.Query("alpha:a1:somedata")
	require.NoError(t, err)
	assert.True(t, q1 == q2, "same query string must return identical query objects")

	q3, err := ds.Query("alpha:a1:otherdata")
	require.NoError(t, err)
	assert.True(t, q1 != q3, "different query string must return different query objects")
}

func TestDomains_ConcurrentAdd(t *testing.T) {
	ds := korrel8r.NewDomains()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			ds.Add(mock.NewDomain(fmt.Sprintf("d%d", i), "c"))
		})
	}
	wg.Wait()
	assert.Len(t, ds.List(), 100)
}

func TestDomains_ConcurrentQuery(t *testing.T) {
	ds := newDomains()
	queries := []string{"alpha:a1:data1", "alpha:a2:data2", "beta:b1:data3"}
	var wg sync.WaitGroup
	results := make([]korrel8r.Query, len(queries))
	for i, qs := range queries {
		wg.Go(func() {
			q, err := ds.Query(qs)
			require.NoError(t, err)
			results[i] = q
		})
	}
	wg.Wait()

	for i, qs := range queries {
		q, err := ds.Query(qs)
		require.NoError(t, err)
		assert.True(t, results[i] == q, "cached query must return identical object for %v", qs)
	}
}

func TestDomains_ConcurrentMixedOps(t *testing.T) {
	ds := korrel8r.NewDomains()
	ds.Add(mock.NewDomain("base", "c1", "c2"))

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			ds.Add(mock.NewDomain(fmt.Sprintf("x%d", i), "c"))
		})
		wg.Go(func() {
			ds.List()
		})
		wg.Go(func() {
			_, _ = ds.Domain("base")
		})
		wg.Go(func() {
			_, _ = ds.Query("base:c1:data")
		})
	}
	wg.Wait()
}
