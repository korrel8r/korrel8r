// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDomains() korrel8r.Domains {
	ds := korrel8r.Domains{}
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
