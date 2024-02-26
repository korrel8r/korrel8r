// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
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

type mockStoreErrorDomain struct{ mock.Domain }

func (d mockStoreErrorDomain) Store(sc korrel8r.StoreConfig) (korrel8r.Store, error) {
	return nil, fmt.Errorf("bad store: %v", sc)
}

func TestApply_bad_stores(t *testing.T) {
	foo, bar := mockStoreErrorDomain{Domain: "foo"}, mock.Domain("bar")
	c := Configs{
		"a": &Config{Stores: []korrel8r.StoreConfig{{"domain": "foo", "x": "y"}}},
		"b": &Config{Stores: []korrel8r.StoreConfig{{"domain": "bar", "a": "b"}}},
	}
	e := engine.New(foo, bar)
	require.NoError(t, c.Apply(e))
	// Check for expected errors
	assert.Equal(t, "bad store: map[domain:foo x:y]", e.StoreConfigsFor(foo)[0][korrel8r.StoreKeyError])
	// Check that we did apply the good stores
	assert.Empty(t, e.StoresFor(foo))
	assert.Equal(t, mock.NewStore(bar, korrel8r.StoreConfig{"domain": "bar", "a": "b"}), e.StoresFor(bar)[0])
}

// TestApply_store_templates tests that we can use templates in store declarations.
// Used in default configurations to get route hosts as part of the default URL.
func TestApply_store_templates(t *testing.T) {
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	c := Configs{
		"a": &Config{Stores: []korrel8r.StoreConfig{{"domain": "foo", "x": `{{ "fooStore" }}`}}},
		"b": &Config{Stores: []korrel8r.StoreConfig{{"domain": "bar", "y": `{{ get "foo:a:[1,2,3]" }}`}}},
	}
	e := engine.New(foo, bar)
	require.NoError(t, c.Apply(e))
	// Check there are no errors
	assert.Empty(t, e.StoreConfigsFor(foo)[0][korrel8r.StoreKeyError])
	assert.Empty(t, e.StoreConfigsFor(bar)[0][korrel8r.StoreKeyError])
	// Check the stores are as expeced
	assert.Equal(t, mock.NewStore(foo, korrel8r.StoreConfig{"domain": "foo", "x": "fooStore"}), e.StoresFor(foo)[0])
	assert.Equal(t, mock.NewStore(bar, korrel8r.StoreConfig{"domain": "bar", "y": "[1 2 3]"}), e.StoresFor(bar)[0])
}
