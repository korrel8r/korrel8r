package config_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_More(t *testing.T) {
	c, err := config.Load("testdata/config.json")
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

func TestMerge_ExpandGroups(t *testing.T) {
	configs := config.Configs{
		"test": &api.Config{
			Groups: []api.Group{
				{Name: "x", Domain: "foo", Classes: []string{"p", "q"}},
				{Name: "y", Domain: "foo", Classes: []string{"x", "a"}},
				{Name: "z", Domain: "foo", Classes: []string{"u", "v"}},
			},
			Rules: []api.Rule{
				{
					Name:   "r",
					Start:  api.ClassSpec{Domain: "foo", Classes: []string{"y"}},
					Goal:   api.ClassSpec{Domain: "foo", Classes: []string{"z"}},
					Result: api.ResultSpec{Query: "dummy"},
				},
			},
		},
	}
	foo := mock.Domain("foo")
	e := engine.New(foo)
	require.NoError(t, configs.Apply(e))
	assert.Equal(t, []mock.Rule{
		mock.NewRule("r", foo.Class("p"), foo.Class("u")),
		mock.NewRule("r", foo.Class("p"), foo.Class("v")),
		mock.NewRule("r", foo.Class("q"), foo.Class("u")),
		mock.NewRule("r", foo.Class("q"), foo.Class("v")),
		mock.NewRule("r", foo.Class("a"), foo.Class("u")),
		mock.NewRule("r", foo.Class("a"), foo.Class("v")),
	}, mock.NewRules(e.Rules()...))
}

func TestMerge_SameGroupDifferentDomain(t *testing.T) {
	configs := config.Configs{
		"test": &api.Config{
			Groups: []api.Group{
				{Name: "x", Domain: "foo", Classes: []string{"p", "q"}},
				{Name: "x", Domain: "bar", Classes: []string{"bbq"}},
			},
			Rules: []api.Rule{
				{
					Name:   "r1",
					Start:  api.ClassSpec{Domain: "foo", Classes: []string{"a", "x"}},
					Goal:   api.ClassSpec{Domain: "bar", Classes: []string{"x"}},
					Result: api.ResultSpec{Query: "help"},
				},
			},
		},
	}
	foo, bar := mock.Domain("foo"), mock.Domain("bar")
	e := engine.New(foo, bar)
	require.NoError(t, configs.Apply(e))
	assert.Equal(t, []mock.Rule{
		mock.NewRule("r1", foo.Class("a"), bar.Class("bbq")),
		mock.NewRule("r1", foo.Class("p"), bar.Class("bbq")),
		mock.NewRule("r1", foo.Class("q"), bar.Class("bbq")),
	}, mock.NewRules(e.Rules()...))
}

func TestLoad_Default(t *testing.T) {
	test.SkipIfNoCluster(t)
	config, err := config.Load("../../korrel8r.yaml")
	require.NoError(t, err)
	e := engine.New(k8s.Domain, alert.Domain, log.Domain, metric.Domain)
	require.NoError(t, config.Apply(e))
	assert.Len(t, e.StoresFor(k8s.Domain), 1)
}
