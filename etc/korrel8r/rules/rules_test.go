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

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
	"github.com/korrel8r/korrel8r/pkg/engine"
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
	configs := test.Must(config.Load("../korrel8r.yaml"))
	for _, c := range configs {
		c.Stores = nil // Use fake stores, not configured defaults.
	}
	c := fake.NewClientBuilder().WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(k8s.Scheme)).Build()
	e, err := engine.Build().
		Domains(k8s.Domain, log.Domain, netflow.Domain, alert.Domain, metric.Domain).
		Apply(configs).
		Stores(test.Must(k8s.NewStore(c, &rest.Config{}))).Engine()
	test.PanicErr(err)
	return e
}

func testTraverse(t *testing.T, e *engine.Engine, start, goal korrel8r.Class, starters []korrel8r.Object, want korrel8r.Query) {
	t.Helper()
	g := e.Graph()
	g.NodeFor(start).Result.Append(starters...)
	f := e.Follower(context.Background(), nil)
	g = g.Traverse(start, []korrel8r.Class{goal}, func(l *graph.Line) bool {
		f.Traverse(l)
		if len(l.Queries) > 0 { // Only consider the rule used if it generated some queries
			tested(l.Rule.Name())
		}
		return true
	})
	assert.NoError(t, f.Err)
	assert.Contains(t, g.NodeFor(goal).Queries, want.String())
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

var rules = unique.Set[string]{}
