// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDomains() *korrel8r.Domains {
	ds := korrel8r.NewDomains()
	for _, d := range domains.All {
		ds.Add(d)
	}
	return ds
}

func newRules(t testing.TB) []korrel8r.Rule { return Rules(testDomains()) }

func ruleByName(t testing.TB, rules []korrel8r.Rule, name string) korrel8r.Rule {
	t.Helper()
	for _, r := range rules {
		if r.Name() == name {
			return r
		}
	}
	t.Fatalf("rule not found: %v", name)
	return nil
}

func queriesToStrings(queries []korrel8r.Query) []string {
	result := make([]string, 0, len(queries))
	for _, q := range queries {
		result = append(result, q.String())
	}
	return result
}

func TestRules(t *testing.T) {
	for _, x := range []struct {
		name  string
		start korrel8r.Object
		want  []string
	}{
		// Standard Prometheus labels
		{
			name:  "MetricToPod",
			start: metric.Object{Labels: map[string]string{"namespace": "foo", "pod": "bar"}},
			want:  []string{`k8s:Pod.v1:{"namespace":"foo","name":"bar"}`},
		},
		// OTel labels
		{
			name:  "MetricToPod",
			start: metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_pod_name": "bar"}},
			want:  []string{`k8s:Pod.v1:{"namespace":"foo","name":"bar"}`},
		},
		{
			name:  "MetricToDeployment",
			start: metric.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}},
			want:  []string{`k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
		{
			name:  "MetricToDeployment",
			start: metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_deployment_name": "bar"}},
			want:  []string{`k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
		{
			name:  "AlertToDeployment",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}},
			want:  []string{`k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
	} {
		t.Run(fmt.Sprintf("%v(%v)", x.name, x.start), func(t *testing.T) {
			r := ruleByName(t, newRules(t), x.name)
			got, err := r.Apply(x.start)
			require.NoError(t, err)
			assert.Equal(t, x.want, queriesToStrings(got))
		})
	}
}

// TestRulesRequired verifies that missing required fields produce an error, like the
// required template function in configuration-file rules.
func TestRulesRequired(t *testing.T) {
	for _, x := range []struct {
		name  string
		start korrel8r.Object
	}{
		{name: "MetricToPod", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
		{name: "MetricToPod", start: metric.Object{Labels: map[string]string{"pod": "bar"}}},
		{name: "MetricToDeployment", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
		{name: "AlertToDeployment", start: &alert.Object{Labels: map[string]string{}}},
	} {
		r := ruleByName(t, newRules(t), x.name)
		queries, err := r.Apply(x.start)
		assert.Error(t, err)
		assert.Empty(t, queries)
	}
}

// TestRuleInterfaces verifies a compiled rule satisfies the korrel8r.Rule interface methods.
func TestRuleInterfaces(t *testing.T) {
	r := ruleByName(t, newRules(t), "MetricToPod")
	assert.Equal(t, "MetricToPod", r.Name())
	require.Len(t, r.Start(), 1)
	require.Len(t, r.Goal(), 1)
	assert.Equal(t, "metric:metric", r.Start()[0].String())
	assert.Equal(t, "k8s:Pod.v1", r.Goal()[0].String())
}

// TestParseRuleAnnotations verifies the registry metadata is read from the *.qtpl annotations.
func TestParseRuleAnnotations(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		specs, err := parseRuleAnnotations(`{% package quickrules %}
{% import "example.com/foo" %}
# Module comment, not part of any rule.
# Foo creates k8s Pod queries from metrics.
name: Foo
start:
  domain: metric
  classes: [metric]
goal:
  domain: k8s
  classes: [Pod]

{% func Foo(o interface{}) %}{% endfunc %}
name: Bar
start:
  domain: alert
  classes: [alert]
goal:
  domain: k8s
  classes: [Deployment.apps]

{% func Bar(o interface{}) %}{% endfunc %}
`)
		require.NoError(t, err)
		require.Len(t, specs, 2)
		assert.Equal(t, []*ruleSpec{
			{Name: "Foo", Start: classSpec{Domain: "metric", Classes: []string{"metric"}}, Goal: classSpec{Domain: "k8s", Classes: []string{"Pod"}}},
			{Name: "Bar", Start: classSpec{Domain: "alert", Classes: []string{"alert"}}, Goal: classSpec{Domain: "k8s", Classes: []string{"Deployment.apps"}}},
		}, specs)
	})
	for _, x := range []struct {
		name, src, wantErr string
	}{
		{"duplicate annotation", `name: Foo
start: {domain: metric, classes: [metric]}
goal: {domain: k8s, classes: [Pod]}

{% func Foo(o interface{}) %}{% endfunc %}
name: Foo
start: {domain: metric, classes: [metric]}
goal: {domain: k8s, classes: [Pod]}

{% func Foo(o interface{}) %}{% endfunc %}`, "duplicate"},
		{"func without annotation", `{% func Foo(o interface{}) %}{% endfunc %}`, "no rule annotation"},
		{"annotation without func", `name: Foo
start: {domain: metric, classes: [metric]}
goal: {domain: k8s, classes: [Pod]}

`, "no matching"},
		{"missing domain", `name: Foo
start: {classes: [metric]}
goal: {domain: k8s, classes: [Pod]}

{% func Foo(o interface{}) %}{% endfunc %}`, "start and goal domains"},
		{"empty classes", `name: Foo
start: {domain: metric, classes: [metric]}
goal: {domain: k8s}

{% func Foo(o interface{}) %}{% endfunc %}`, "start and goal classes"},
		{"invalid yaml", `name: Foo
start: {classes: [}
goal: {domain: k8s, classes: [Pod]}

{% func Foo(o interface{}) %}{% endfunc %}`, "invalid rule annotation"},
	} {
		t.Run(x.name, func(t *testing.T) {
			_, err := parseRuleAnnotations(x.src)
			require.Error(t, err)
			assert.Contains(t, err.Error(), x.wantErr)
		})
	}
}

// BenchmarkRules compares a compiled rule with the equivalent Go-template rule.
func BenchmarkRules(b *testing.B) {
	d := testDomains()
	compiled := ruleByName(b, newRules(b), "MetricToPod")
	templateRule := newTemplateMetricToPod(b, d)
	start := metric.Object{Labels: map[string]string{"namespace": "foo", "pod": "bar"}}
	b.Run("template", func(b *testing.B) { benchmarkRule(b, templateRule, start) })
	b.Run("quick", func(b *testing.B) { benchmarkRule(b, compiled, start) })
}

func benchmarkRule(b *testing.B, r korrel8r.Rule, start korrel8r.Object) {
	b.Helper()
	for b.Loop() {
		queries, err := r.Apply(start)
		if err != nil || len(queries) != 1 || !strings.HasPrefix(queries[0].String(), "k8s:Pod") {
			b.Fatal(err)
		}
	}
}

// newTemplateMetricToPod creates the equivalent Go-template rule for benchmarking.
func newTemplateMetricToPod(b *testing.B, d *korrel8r.Domains) korrel8r.Rule {
	b.Helper()
	start, err := d.Class("metric:metric")
	require.NoError(b, err)
	goal, err := d.Class("k8s:Pod")
	require.NoError(b, err)
	tmpl := template.New("MetricToPod").Funcs(template.FuncMap{
		"required": func(s string) (string, error) {
			if s == "" {
				return "", errors.New("required")
			}
			return s, nil
		},
	})
	r, err := rules.NewTemplateRule([]korrel8r.Class{start}, []korrel8r.Class{goal}, tmpl,
		`k8s:Pod:{"namespace":"{{or (index .Labels "k8s_namespace_name") (index .Labels "namespace") | required}}","name":"{{or (index .Labels "k8s_pod_name") (index .Labels "pod") | required}}"}`,
		d)
	require.NoError(b, err)
	return r
}
