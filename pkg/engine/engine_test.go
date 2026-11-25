// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func list[T any](x ...T) []T { return x }

func TestRefreshNewClasses(t *testing.T) {
	d := mock.NewGrowingDomain("g")
	a, b, c := d.Class("a"), d.Class("b"), d.Class("c")
	s := mock.NewDomain("s")
	x, y := s.Class("x"), s.Class("y")
	cfg := config.Configs{{
		Rules: []config.Rule{{Name: "config-ba",
			Start:  config.ClassSpec{Domain: "g", Classes: []string{"b"}},
			Goal:   config.ClassSpec{Domain: "g", Classes: []string{"a"}},
			Result: config.ResultSpec{Query: "g:a:x"}}},
		StatusRules: []config.StatusRule{{Name: "config-status",
			Start: config.ClassSpec{Domain: "g", Classes: []string{"b"}}, Status: "ok"}},
	}}
	var factoryCalls atomic.Int64
	e, err := engine.Build().Domains(d).Config(cfg).
		Rules(mock.NewRule("xy", list(x), list(y), mock.NewQuery(y, ""))).
		StatusRules(status.New(list(a), template.Must(template.New("static").Parse("ok")))).
		RuleFactory(func(*korrel8r.Domains) []korrel8r.Rule {
			factoryCalls.Add(1)
			if d.ClassVersion() == 0 {
				return nil
			}
			return list[korrel8r.Rule](mock.NewRule("bc", list(b), list(c), mock.NewQuery(c, "")))
		}).Engine()
	require.NoError(t, err)
	assert.EqualValues(t, 1, factoryCalls.Load())
	assert.Nil(t, e.Rule("bc"))

	// No new classes, no rebuild.
	old := e.GraphData()
	require.NoError(t, e.RefreshNewClasses())
	assert.Same(t, old, e.GraphData())
	assert.EqualValues(t, 1, factoryCalls.Load())

	// New classes rebuild once, even with concurrent searches.
	d.Grow()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { assert.NoError(t, e.RefreshNewClasses()) })
	}
	wg.Wait()
	assert.EqualValues(t, 2, factoryCalls.Load())
	for _, name := range []string{"xy", "config-ba", "bc"} {
		assert.NotNil(t, e.Rule(name), name)
	}
	assert.Len(t, e.StatusRulesFor(a), 1)
	assert.Len(t, e.StatusRulesFor(b), 1)
	assert.NotNil(t, e.ClassMetricAttrs(c))
	assert.NotNil(t, e.RuleMetricAttrs(e.Rule("bc")))
	assert.Equal(t, 2, old.LineCount()) // Old searches retain their topology.
	assert.Equal(t, 3, e.GraphData().LineCount())
}

func TestRuleAccessorsRefresh(t *testing.T) {
	for name, refreshed := range map[string]func(*engine.Engine, korrel8r.Class) bool{
		"Rule":           func(e *engine.Engine, _ korrel8r.Class) bool { return e.Rule("new") != nil },
		"Rules":          func(e *engine.Engine, _ korrel8r.Class) bool { return len(e.Rules()) == 2 },
		"GraphData":      func(e *engine.Engine, _ korrel8r.Class) bool { return e.GraphData().LineCount() == 2 },
		"Graph":          func(e *engine.Engine, _ korrel8r.Class) bool { return e.Graph().Nodes().Len() == 3 },
		"StatusRulesFor": func(e *engine.Engine, b korrel8r.Class) bool { return len(e.StatusRulesFor(b)) == 1 },
	} {
		t.Run(name, func(t *testing.T) {
			d := mock.NewGrowingDomain("g")
			a, b, c := d.Class("a"), d.Class("b"), d.Class("c")
			e, err := engine.Build().Domains(d).
				RuleFactory(func(*korrel8r.Domains) []korrel8r.Rule {
					if d.ClassVersion() == 0 {
						return []korrel8r.Rule{mock.NewRule("old", list(a), list(b), mock.NewQuery(b, ""))}
					}
					return []korrel8r.Rule{ // Adds a node to the graph.
						mock.NewRule("new", list(a), list(b), mock.NewQuery(b, "")),
						mock.NewRule("bc", list(b), list(c), mock.NewQuery(c, "")),
					}
				}).
				StatusRuleFactory(func(*korrel8r.Domains) []status.Rule {
					if d.ClassVersion() == 0 {
						return nil
					}
					return []status.Rule{status.New(list(b), template.Must(template.New("s").Parse("ok")))}
				}).Engine()
			require.NoError(t, err)
			oldRule := e.Rule("old")
			require.NotNil(t, oldRule)
			d.Grow()
			assert.True(t, refreshed(e, b)) // No explicit RefreshNewClasses.
			// Rules from the previous state still have metric attributes, for searches in progress.
			assert.Nil(t, e.Rule("old"))
			assert.NotNil(t, e.RuleMetricAttrs(oldRule))
		})
	}
}

func TestRefreshNewClassesError(t *testing.T) {
	d := mock.NewGrowingDomain("g")
	a, b := d.Class("a"), d.Class("b")
	var calls atomic.Int64
	e, err := engine.Build().Domains(d).
		RuleFactory(func(*korrel8r.Domains) []korrel8r.Rule {
			calls.Add(1)
			rules := []korrel8r.Rule{mock.NewRule("ab", list(a), list(b), mock.NewQuery(b, ""))}
			switch d.ClassVersion() {
			case 0:
				return rules
			case 1: // Invalid: duplicate rule name.
				return append(rules, mock.NewRule("ab", list(b), list(a), mock.NewQuery(a, "")))
			default:
				return append(rules, mock.NewRule("ba", list(b), list(a), mock.NewQuery(a, "")))
			}
		}).Engine()
	require.NoError(t, err)
	old := e.GraphData()

	// A failed rebuild keeps the previous rules.
	d.Grow()
	require.Error(t, e.RefreshNewClasses())
	assert.Same(t, old, e.GraphData())
	assert.NotNil(t, e.Rule("ab"))
	assert.EqualValues(t, 2, calls.Load())

	// No retry until classes change again.
	require.NoError(t, e.RefreshNewClasses())
	assert.EqualValues(t, 2, calls.Load())

	// New classes trigger another rebuild.
	d.Grow()
	require.NoError(t, e.RefreshNewClasses())
	assert.EqualValues(t, 3, calls.Load())
	assert.NotNil(t, e.Rule("ba"))
}

func TestRuleFactories(t *testing.T) {
	d := mock.NewDomain("factory", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	ruleCalls, statusCalls := 0, 0
	e, err := engine.Build().Domains(d).
		RuleFactory(func(domains *korrel8r.Domains) []korrel8r.Rule {
			ruleCalls++
			assert.Same(t, d, domains.Get("factory"))
			return []korrel8r.Rule{mock.NewRule("ab", list(a), list(b), mock.NewQuery(b, ""))}
		}).
		StatusRuleFactory(func(domains *korrel8r.Domains) []status.Rule {
			statusCalls++
			assert.Same(t, d, domains.Get("factory"))
			return []status.Rule{status.New(list(a), template.Must(template.New("label").Parse("ok")))}
		}).
		RuleFactory(func(*korrel8r.Domains) []korrel8r.Rule {
			return []korrel8r.Rule{mock.NewRule("ba", list(b), list(a), mock.NewQuery(a, ""))}
		}).
		StatusRuleFactory(func(*korrel8r.Domains) []status.Rule {
			return []status.Rule{status.New(list(b), template.Must(template.New("other").Parse("ok")))}
		}).Engine()
	require.NoError(t, err)
	assert.Equal(t, 1, ruleCalls)
	assert.Equal(t, 1, statusCalls)
	assert.NotNil(t, e.Rule("ab"))
	assert.NotNil(t, e.Rule("ba"))
	assert.Len(t, e.StatusRulesFor(a), 1)
	assert.Len(t, e.StatusRulesFor(b), 1)
}

func TestRefreshPreviouslyMissingRule(t *testing.T) {
	const class = "Widget.v1.refresh.example.io"
	cfg := config.Configs{{Rules: []config.Rule{{
		Name:   "pod-to-refresh-widget",
		Start:  config.ClassSpec{Domain: "k8s", Classes: []string{"Pod"}},
		Goal:   config.ClassSpec{Domain: "k8s", Classes: []string{class}},
		Result: config.ResultSpec{Query: `k8s:Widget.v1.refresh.example.io:{}`},
	}}}}
	ruleCalls, statusCalls := 0, 0
	e, err := engine.Build().Domains(k8s.Domain).Config(cfg).
		RuleFactory(func(domains *korrel8r.Domains) []korrel8r.Rule {
			ruleCalls++
			widget := domains.Get("k8s").Class(class)
			if widget == nil {
				return nil
			}
			pod := domains.Get("k8s").Class("Pod")
			return []korrel8r.Rule{mock.NewRule("factory-widget", list(pod), list(widget), mock.NewQuery(widget, ""))}
		}).
		StatusRuleFactory(func(domains *korrel8r.Domains) []status.Rule {
			statusCalls++
			widget := domains.Get("k8s").Class(class)
			if widget == nil {
				return nil
			}
			return []status.Rule{status.New(list(widget), template.Must(template.New("widget").Parse("ok")))}
		}).Engine()
	require.NoError(t, err)
	assert.Equal(t, 1, ruleCalls)
	assert.Equal(t, 1, statusCalls)
	other, err := engine.Build().Domains(k8s.Domain).Config(cfg).Engine()
	require.NoError(t, err)
	assert.Nil(t, e.Rule("pod-to-refresh-widget"))
	old := e.GraphData()
	_, err = k8s.Domain.NewStoreWithDiscovery(fake.NewClientBuilder().Build(), &rest.Config{},
		&fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{Resources: []*metav1.APIResourceList{{
			GroupVersion: "refresh.example.io/v1", APIResources: []metav1.APIResource{{Name: "widgets", Kind: "Widget", Namespaced: true}},
		}}}})
	require.NoError(t, err)
	require.NoError(t, e.RefreshNewClasses())
	require.NoError(t, other.RefreshNewClasses()) // Independent indexes for concurrent engines.
	assert.NotNil(t, other.Rule("pod-to-refresh-widget"))
	assert.Equal(t, 0, old.LineCount())
	assert.Equal(t, 2, e.GraphData().LineCount())
	assert.Equal(t, 2, ruleCalls)
	assert.Equal(t, 2, statusCalls)
	assert.NotNil(t, e.Rule("factory-widget"))
	assert.Len(t, e.StatusRulesFor(k8s.Domain.Class(class)), 1)
	assert.NotNil(t, e.Rule("pod-to-refresh-widget"))
}

func TestEngine_Class(t *testing.T) {
	domain := mock.NewDomain("mock")
	foo, bar := domain.Class("foo"), domain.Class("bar")
	rule := mock.NewRule("x", list(foo), list(bar), mock.NewQuery(bar, ""))
	for _, x := range []struct {
		name string
		want korrel8r.Class
		err  string
	}{
		{"mock:foo", foo, ""},
		{"x.", nil, "invalid class name: x."},
		{".x", nil, "invalid class name: .x"},
		{"x", nil, "invalid class name: x"},
		{"", nil, "invalid class name: "},
		{"bad:foo", nil, `domain not found: bad`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e, err := engine.Build().Rules(rule).Engine()
			require.NoError(t, err)
			assert.Equal(t, []korrel8r.Domain{domain}, e.Domains())
			c, err := e.Class(x.name)
			if x.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, x.err)
			}
			assert.Equal(t, x.want, c)
		})
	}
}

func (o obj) String() string { return o.Name }

func TestEngine_PropagateConstraints(t *testing.T) {
	d := mock.NewDomain("mock")
	a, b, c := d.Class("a"), d.Class("b"), d.Class("c")
	// Time range [start,end] and some time points.
	start := time.Unix(0, 0)
	end := start.Add(time.Minute)
	afterEnd := end.Add(time.Minute)
	early, ontime, late := start.Add(-1), start.Add(1), end.Add(1)
	s := mock.NewStore(d, a, b, c)

	s.ConstraintFunc = func(c *korrel8r.Constraint, o korrel8r.Object) bool {
		return c.CompareTime(o.(obj).Time) == 0
	}

	// Rules to test constraints.
	e, err := engine.Build().Rules(
		// Generate objects with timepoints.
		mock.NewRule("ab", []korrel8r.Class{a}, []korrel8r.Class{b}, func(korrel8r.Object) ([]korrel8r.Query, error) {
			return []korrel8r.Query{s.NewQuery(b, obj{"x", early}, obj{"y", ontime}, obj{"z", late})}, nil
		}),
		// Generate objects with timeponts and record name of previous object.
		mock.NewRule("bc", []korrel8r.Class{b}, []korrel8r.Class{c}, func(o korrel8r.Object) ([]korrel8r.Query, error) {
			name := o.(obj).Name
			return []korrel8r.Query{s.NewQuery(c, obj{"u" + name, early}, obj{"v" + name, ontime}, obj{"w" + name, late})}, nil

		})).Stores(s).Engine()
	require.NoError(t, err)

	// Test traversals, verify constraints are applied.
	for i, x := range []struct {
		constraint *korrel8r.Constraint
		want       []obj
	}{
		{&korrel8r.Constraint{Start: &start, End: &end}, []obj{
			{"vy", ontime},
		}},
		{&korrel8r.Constraint{End: &afterEnd}, []obj{
			{"ux", early}, {"vx", ontime}, {"wx", late},
			{"uy", early}, {"vy", ontime}, {"wy", late},
			{"uz", early}, {"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{Start: &start, End: &afterEnd}, []obj{
			{"vy", ontime}, {"wy", late},
			{"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{End: &end}, []obj{
			{"ux", early}, {"vx", ontime},
			{"uy", early}, {"vy", ontime},
		}},
	} {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			goals := []korrel8r.Class{c}
			start := traverse.Start{Class: a, Objects: []korrel8r.Object{obj{"a", ontime}}, Constraint: x.constraint}
			g, err := traverse.Goals(context.Background(), e, start, goals)
			assert.NoError(t, err)
			if node := g.NodeFor(c); assert.NotNil(t, node) {
				got := node.Result.List()
				assert.Equal(t, asStrings(x.want), asStrings(got), "want %v got %v", x.want, got)
			}
		})
	}
}

func TestEngine_ConfigMockStore(t *testing.T) {
	d := mock.NewDomain("mock")
	e, err := engine.Build().Domains(d).ConfigFile("testdata/korrel8r.yaml").Engine()
	require.NoError(t, err)
	q, err := e.Query("mock:foo:hello")
	require.NoError(t, err)
	r := result.New(q.Class())
	require.NoError(t, e.Get(context.Background(), q, nil, r))
	assert.Equal(t, []korrel8r.Object{"hello", "there"}, r.List())
}

func asStrings[T any](v []T) []string {
	s := make([]string, len(v))
	for i := range v {
		s[i] = fmt.Sprint(v[i])
	}
	slices.Sort(s)
	return s
}

func TestEngineStoreFor(t *testing.T) {
	d := mock.NewDomain("mock")
	// add an extra mock store, verify both stores are checked by Get.
	s := mock.NewStore(d)
	q := mock.NewQuery(d.Class("foo"), "hello")
	s.AddQuery(q, "dolly")
	e, err := engine.Build().Domains(d).ConfigFile("testdata/korrel8r.yaml").Stores(s).Engine()
	require.NoError(t, err)

	r := &mock.Result{}
	require.NoError(t, e.StoreFor(d).Get(context.Background(), q, nil, r))
	assert.ElementsMatch(t, []korrel8r.Object{"hello", "there", "dolly"}, r.List())

	r = &mock.Result{}
	q = mock.NewQuery(d.Class("bar"), "help")
	require.NoError(t, e.StoreFor(d).Get(context.Background(), q, nil, r))
	assert.ElementsMatch(t, []korrel8r.Object{"help", "me"}, r.List())
}

func TestEngine_LabelersFor(t *testing.T) {
	d := mock.NewDomain("mock", "a", "b")
	a := d.Class("a")
	b := d.Class("b")
	s := mock.NewStore(d, a, b)
	e, err := engine.Build().Domains(d).Stores(s).
		Config(config.Configs{
			{
				Rules: []config.Rule{{
					Name:   "ab",
					Start:  config.ClassSpec{Domain: "mock", Classes: []string{"a"}},
					Goal:   config.ClassSpec{Domain: "mock", Classes: []string{"b"}},
					Result: config.ResultSpec{Query: `mock:b:x`},
				}},
				StatusRules: []config.StatusRule{
					{
						Name:   "label-a",
						Start:  config.ClassSpec{Domain: "mock", Classes: []string{"a"}},
						Status: `label-from-a`,
					},
					{
						Name:   "label-a2",
						Start:  config.ClassSpec{Domain: "mock", Classes: []string{"a"}},
						Status: `another-label`,
					},
				},
			},
		}).Engine()
	require.NoError(t, err)

	// Class a should have 2 labelers
	statuses := e.StatusRulesFor(a)
	assert.Len(t, statuses, 2)
	assert.Equal(t, "label-a", statuses[0].Name())
	assert.Equal(t, "label-a2", statuses[1].Name())

	// Class b should have no status rules
	assert.Empty(t, e.StatusRulesFor(b))
}

func TestEngine_NamedTemplates(t *testing.T) {
	d := mock.NewDomain("mock", "a", "b")
	a := d.Class("a")
	b := d.Class("b")
	s := mock.NewStore(d, a, b)
	q := mock.NewQuery(b, "hello")
	s.AddQuery(q, "world")

	e, err := engine.Build().Domains(d).Stores(s).
		Config(config.Configs{
			{
				Templates: []config.Template{
					{Name: "helper", Template: `mock:b:hello`},
				},
				Rules: []config.Rule{{
					Name:   "ab",
					Start:  config.ClassSpec{Domain: "mock", Classes: []string{"a"}},
					Goal:   config.ClassSpec{Domain: "mock", Classes: []string{"b"}},
					Result: config.ResultSpec{Query: `{{template "helper" .}}`},
				}},
			},
		}).Engine()
	require.NoError(t, err)

	// Verify rule applies using the named template
	g, err := traverse.Goals(context.Background(), e,
		traverse.Start{Class: a, Objects: []korrel8r.Object{"x"}},
		[]korrel8r.Class{b})
	require.NoError(t, err)
	if node := g.NodeFor(b); assert.NotNil(t, node) {
		assert.Equal(t, []korrel8r.Object{"world"}, node.Result.List())
	}
}

func TestEngine_NamedTemplates_FromInclude(t *testing.T) {
	d := mock.NewDomain("mock", "a", "b")
	a := d.Class("a")
	b := d.Class("b")
	s := mock.NewStore(d, a, b)
	q := mock.NewQuery(b, "hello")
	s.AddQuery(q, "world")

	// Templates from one config should be available in rules from another config.
	e, err := engine.Build().Domains(d).Stores(s).
		Config(config.Configs{
			{
				Templates: []config.Template{
					{Name: "shared", Template: `mock:b:hello`},
				},
			},
			{
				Rules: []config.Rule{{
					Name:   "ab",
					Start:  config.ClassSpec{Domain: "mock", Classes: []string{"a"}},
					Goal:   config.ClassSpec{Domain: "mock", Classes: []string{"b"}},
					Result: config.ResultSpec{Query: `{{template "shared" .}}`},
				}},
			},
		}).Engine()
	require.NoError(t, err)

	g, err := traverse.Goals(context.Background(), e,
		traverse.Start{Class: a, Objects: []korrel8r.Object{"x"}},
		[]korrel8r.Class{b})
	require.NoError(t, err)
	if node := g.NodeFor(b); assert.NotNil(t, node) {
		assert.Equal(t, []korrel8r.Object{"world"}, node.Result.List())
	}
}

func TestEngine_NamedTemplates_Invalid(t *testing.T) {
	d := mock.NewDomain("mock", "a", "b")
	_, err := engine.Build().Domains(d).
		Config(config.Configs{
			{
				Templates: []config.Template{
					{Name: "bad", Template: `{{invalid`},
				},
			},
		}).Engine()
	assert.ErrorContains(t, err, `invalid template "bad"`)
}

// Mock object has a name and a timestamp.
type obj struct {
	Name string
	Time time.Time
}
