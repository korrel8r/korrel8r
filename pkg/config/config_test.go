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

	e, err := engine.Build().Domains(foo, bar).Apply(c).Engine()
	require.NoError(t, err)
	assert.Equal(t, []*mock.Rule{
		mock.NewRuleMulti("rule1",
			[]korrel8r.Class{foo.Class("a"), foo.Class("b"), foo.Class("d"), foo.Class("e")},
			[]korrel8r.Class{bar.Class("z")}),
		mock.NewRuleMulti("rule2",
			[]korrel8r.Class{foo.Class("d"), foo.Class("e")},
			[]korrel8r.Class{bar.Class("q")}),
	}, mock.NewRules(e.Rules()...))
}

func TestApply_ExpandGroups(t *testing.T) {
	c := Configs{
		"test": &Config{
			Aliases: []Class{
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
	e, err := engine.Build().Domains(foo).Apply(c).Engine()
	require.NoError(t, err)
	assert.Equal(t, []*mock.Rule{
		mock.NewRuleMulti("r",
			[]korrel8r.Class{foo.Class("p"), foo.Class("q"), foo.Class("a")},
			[]korrel8r.Class{foo.Class("u"), foo.Class("v")}),
	}, mock.NewRules(e.Rules()...))
}

func TestApply_SameGroupDifferentDomain(t *testing.T) {
	c := Configs{
		"test": &Config{
			Aliases: []Class{
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
	e, err := engine.Build().Domains(foo, bar).Apply(c).Engine()
	require.NoError(t, err)
	assert.Equal(t, []*mock.Rule{
		mock.NewRuleMulti("r1",
			[]korrel8r.Class{foo.Class("a"), foo.Class("p"), foo.Class("q")},
			[]korrel8r.Class{bar.Class("bbq")}),
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
	e, err := engine.Build().Domains(foo, bar).Apply(c).Engine()
	require.NoError(t, err)
	// Check for expected errors
	assert.Equal(t,
		korrel8r.StoreConfig{"domain": "foo", "x": "y", "error": "bad store: map[domain:foo x:y]", "errorCount": "1"},
		e.StoreConfigsFor(foo)[0])
	// Check that we did apply the good stores
	assert.Equal(t, korrel8r.StoreConfig{"a": "b", "domain": "bar"}, e.StoreConfigsFor(bar)[0])
}

// TestApply_store_templates tests that we can use templates in store declarations.
// Used in default configurations to get route hosts as part of the default URL.
func TestApply_store_templates(t *testing.T) {
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	c := Configs{
		"a": &Config{Stores: []korrel8r.StoreConfig{{"domain": "foo", "x": `{{ "fooStore" }}`}}},
		"b": &Config{Stores: []korrel8r.StoreConfig{{"domain": "bar", "y": `{{ get "foo:a:[1,2,3]" }}`}}},
	}
	e, err := engine.Build().Domains(foo, bar).Apply(c).Engine()
	require.NoError(t, err)
	// Check there are no errors
	assert.Empty(t, e.StoreConfigsFor(foo)[0][korrel8r.StoreKeyError])
	assert.Empty(t, e.StoreConfigsFor(bar)[0][korrel8r.StoreKeyError])
	// Check the stores are as expeced
	assert.Equal(t, korrel8r.StoreConfig{"domain": "foo", "x": "fooStore"}, e.StoreConfigsFor(foo)[0])
	assert.Equal(t, korrel8r.StoreConfig{"domain": "bar", "y": "[1 2 3]"}, e.StoreConfigsFor(bar)[0])
}
