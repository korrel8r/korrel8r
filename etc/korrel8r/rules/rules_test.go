// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package rules is a test-only package to unit test YAML rules.
package rules_test

// Test use of rules in graph traversal.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
	"github.com/korrel8r/korrel8r/pkg/domains/trace"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setup() *engine.Engine {
	configs, err := config.Load("all.yaml")
	if err != nil {
		panic(err)
	}
	for _, c := range configs {
		c.Stores = nil // Use fake stores, not configured defaults.
	}
	c := fake.NewClientBuilder().WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(k8s.Scheme)).Build()
	s, err := k8s.NewStore(c, &rest.Config{})
	if err != nil {
		panic(err)
	}
	e, err := engine.Build().
		Domains(k8s.Domain, log.Domain, netflow.Domain, trace.Domain, alert.Domain, metric.Domain).
		Config(configs).
		Stores(s).Engine()
	if err != nil {
		panic(err)
	}
	return e
}

func testTraverse(t *testing.T, e *engine.Engine, start, goal korrel8r.Class, starters []korrel8r.Object, want korrel8r.Query) {
	t.Helper()
	goals := []korrel8r.Class{goal}
	g, err := traverse.NewSync(e, e.Graph(), start, starters, nil).Goals(context.Background(), goals)
	assert.NoError(t, err)
	assert.Contains(t, g.NodeFor(goal).Queries, want.String())
	g.EachLine(func(l *graph.Line) {
		// FIXME stricter test for results?
		if len(l.Queries) > 0 { // Only consider the rule tested if it generated queries
			tested(l.Rule.Name())
		}
	})
}

func TestMain(m *testing.M) {
	e := setup()
	for _, r := range e.Rules() {
		rules.Add(r.Name())
	}
	m.Run()
	if len(rules) > 0 {
		fmt.Printf("FAIL: %v rules not tested:\n- %v\n", len(rules), strings.Join(maps.Keys(rules), "\n- "))
		os.Exit(1)
	}
}

// tested marks a rule as having been tested.
func tested(ruleName string) { rules.Remove(ruleName) }

func apply(e *engine.Engine, ruleName string, start korrel8r.Object) (korrel8r.Query, error) {
	tested(ruleName)
	return e.Rule(ruleName).Apply(start)
}

var rules = unique.Set[string]{}
