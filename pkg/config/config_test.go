// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_More(t *testing.T) {
	c, err := Load("testdata/config.json")
	require.NoError(t, err)
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	e := engine.New(foo, bar)
	require.NoError(t, c.Apply(e))
	assert.Equal(t, []mock.Rule{
		mock.NewRule("rule1", foo.Class("a"), bar.Class("z")),
		mock.NewRule("rule1", foo.Class("b"), bar.Class("z")),
		mock.NewRule("rule1", foo.Class("d"), bar.Class("z")),
		mock.NewRule("rule1", foo.Class("e"), bar.Class("z")),
		mock.NewRule("rule2", foo.Class("d"), bar.Class("q")),
		mock.NewRule("rule2", foo.Class("e"), bar.Class("q")),
	}, mock.NewRules(e.Rules()...))
}

func TestApply_ExpandGroups(t *testing.T) {
	c := Configs{
		"test": &Config{
			Groups: []Group{
				{Name: "x", Domain: "foo", Classes: []string{"p", "q"}},
				{Name: "y", Domain: "foo", Classes: []string{"x", "a"}},
				{Name: "z", Domain: "foo", Classes: []string{"u", "v"}},
			},
			Rules: []Rule{
				{
					Name:   "r",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"y"}},
					Goal:   ClassSpec{Domain: "foo", Classes: []string{"z"}},
					Result: ResultSpec{Query: "dummy"},
				},
			},
		},
	}
	foo := mock.Domain("foo")
	e := engine.New(foo)
	require.NoError(t, c.Apply(e))
	assert.Equal(t, []mock.Rule{
		mock.NewRule("r", foo.Class("p"), foo.Class("u")),
		mock.NewRule("r", foo.Class("p"), foo.Class("v")),
		mock.NewRule("r", foo.Class("q"), foo.Class("u")),
		mock.NewRule("r", foo.Class("q"), foo.Class("v")),
		mock.NewRule("r", foo.Class("a"), foo.Class("u")),
		mock.NewRule("r", foo.Class("a"), foo.Class("v")),
	}, mock.NewRules(e.Rules()...))
}

func TestApply_SameGroupDifferentDomain(t *testing.T) {
	c := Configs{
		"test": &Config{
			Groups: []Group{
				{Name: "x", Domain: "foo", Classes: []string{"p", "q"}},
				{Name: "x", Domain: "bar", Classes: []string{"bbq"}},
			},
			Rules: []Rule{
				{
					Name:   "r1",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"a", "x"}},
					Goal:   ClassSpec{Domain: "bar", Classes: []string{"x"}},
					Result: ResultSpec{Query: "help"},
				},
			},
		},
	}
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	e := engine.New(foo, bar)
	require.NoError(t, c.Apply(e))
	assert.Equal(t, []mock.Rule{
		mock.NewRule("r1", foo.Class("a"), bar.Class("bbq")),
		mock.NewRule("r1", foo.Class("p"), bar.Class("bbq")),
		mock.NewRule("r1", foo.Class("q"), bar.Class("bbq")),
	}, mock.NewRules(e.Rules()...))
}

func TestApply_BadStores(t *testing.T) {
	c := Configs{
		"a": &Config{ Stores: []korrel8r.StoreConfig{{"domain": "foo"},{"domain":"badFoo"}}},
		"b": &Config{ Stores: []korrel8r.StoreConfig{{"domain": "bar"}, {"domain": "badBar"}}},
	}
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	e := engine.New(foo, bar)
	err := c.Apply(e)
	require.IsType(t, (*Status)(nil), err)
	s := err.(*Status)
	require.NoError(t, s.Err)
	assert.Len(t, s.StoreErrs, 2)
	// Check for expected errorss
	assert.Equal(t, korrel8r.StoreConfig{"domain": "badFoo"}, s.StoreErrs[0].Store)
	assert.EqualError(t, s.StoreErrs[0].Err, `a: error creating store {"domain":"badFoo"}: domain not found: "badFoo"`)
	assert.Equal(t, korrel8r.StoreConfig{"domain": "badBar"}, s.StoreErrs[1].Store)
	assert.EqualError(t, s.StoreErrs[1].Err, `b: error creating store {"domain":"badBar"}: domain not found: "badBar"`)
	// Check that we did apply the good stores
	assert.Equal(t, mock.NewStore(foo), e.StoresFor(foo)[0])
	assert.Equal(t, mock.NewStore(bar), e.StoresFor(bar)[0])
}
