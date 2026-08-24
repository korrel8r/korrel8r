// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
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

func newRules(_ testing.TB) []korrel8r.Rule { return Rules(testDomains()) }

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
		{
			name:  "AlertToDeployment",
			start: &alert.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_deployment_name": "bar"}},
			want:  []string{`k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
		{
			name: "DependentToOwner",
			start: k8s.Object{
				"metadata": map[string]any{
					"namespace": "test-ns",
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "ReplicaSet",
							"name":       "my-rs",
						},
					},
				},
			},
			want: []string{`k8s:ReplicaSet.v1.apps:{"namespace":"test-ns","name":"my-rs"}`},
		},
		{
			name: "AllToEvent",
			start: k8s.Object{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   map[string]any{"namespace": "ns1", "name": "pod1"},
			},
			want: []string{`k8s:Event.v1:{"namespace":"ns1","fields":{"involvedObject.apiVersion":"v1","involvedObject.kind":"Pod","involvedObject.name":"pod1","involvedObject.namespace":"ns1"}}`},
		},
		{
			name: "AllToMetric",
			start: k8s.Object{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   map[string]any{"namespace": "ns1", "name": "pod1"},
			},
			want: []string{
				`metric:metric:{namespace="ns1",pod="pod1"}`,
				`metric:metric:{k8s_namespace_name="ns1",k8s_pod_name="pod1"}`,
			},
		},
		{
			name: "SelectorToPods",
			start: k8s.Object{
				"apiVersion": "v1",
				"kind":       "Deployment",
				"metadata":   map[string]any{"namespace": "ns", "name": "x"},
				"spec":       map[string]any{"selector": map[string]any{"matchLabels": map[string]any{"app": "web"}}},
			},
			want: []string{`k8s:Pod.v1:{"namespace":"ns","labels":{"app":"web"}}`},
		},
		{
			name: "SelectorToLogs",
			start: k8s.Object{
				"metadata": map[string]any{"namespace": "ns", "name": "x"},
				"spec":     map[string]any{"selector": map[string]any{"matchLabels": map[string]any{"app": "web"}}},
			},
			want: []string{
				`log:application:{kubernetes_namespace_name="ns"}|json|kubernetes_labels_app="web"`,
			},
		},
		{
			name: "K8sSrcToNetflow",
			start: k8s.Object{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   map[string]any{"namespace": "bar", "name": "foo"},
			},
			want: []string{`netflow:network:{SrcK8S_Type="Pod", SrcK8S_Namespace="bar"} | json | SrcK8S_Name="foo"`},
		},
		{
			name: "K8sSrcOwnerToNetflow",
			start: k8s.Object{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata":   map[string]any{"namespace": "bar", "name": "foo"},
			},
			want: []string{`netflow:network:{SrcK8S_Namespace="bar", SrcK8S_OwnerName="foo"}`},
		},
		{
			name: "K8sDstToNetflow",
			start: k8s.Object{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   map[string]any{"namespace": "bar", "name": "foo"},
			},
			want: []string{`netflow:network:{DstK8S_Type="Pod", DstK8S_Namespace="bar"} | json | DstK8S_Name="foo"`},
		},
		{
			name: "K8sDstOwnerToNetflow",
			start: k8s.Object{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata":   map[string]any{"namespace": "bar", "name": "foo"},
			},
			want: []string{`netflow:network:{DstK8S_Namespace="bar", DstK8S_OwnerName="foo"}`},
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

// TestRulesRequired verifies that missing required fields produce an error.
func TestRulesRequired(t *testing.T) {
	for _, x := range []struct {
		name  string
		start korrel8r.Object
	}{
		{name: "AlertToDeployment", start: &alert.Object{Labels: map[string]string{}}},
		{name: "DependentToOwner", start: k8s.Object{}},
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

// TestRulesUnknownClasses verifies that rules referencing unknown classes (e.g. k8s
// custom resources not present at startup) are handled like configuration rules: unknown
// classes are skipped, and the rule is dropped only if start or goal have no known classes.
func TestRulesUnknownClasses(t *testing.T) {
	d := testDomains()
	// Mixed known/unknown classes: rule is kept with only the known classes.
	r, err := newRule(d, &ruleSpec{
		Name:  "MetricToPod",
		Start: classSpec{Domain: "metric", Classes: []string{"metric"}},
		Goal:  classSpec{Domain: "k8s", Classes: []string{"Pod", "NoSuchResource", "Deployment.apps"}},
	})
	require.NoError(t, err)
	require.NotNil(t, r)
	goal := []string{}
	for _, c := range r.Goal() {
		goal = append(goal, c.String())
	}
	slices.Sort(goal)
	if assert.Len(t, r.Goal(), 2) {
		assert.Equal(t, []string{"k8s:Deployment.v1.apps", "k8s:Pod.v1"}, goal)
	}

	// Only unknown classes in goal: rule is dropped.
	r, err = newRule(d, &ruleSpec{
		Name:  "MetricToPod",
		Start: classSpec{Domain: "metric", Classes: []string{"metric"}},
		Goal:  classSpec{Domain: "k8s", Classes: []string{"NoSuchResource", "AlsoNoSuch"}},
	})
	require.NoError(t, err)
	assert.Nil(t, r)

	// Unknown domain: rule is dropped.
	r, err = newRule(d, &ruleSpec{
		Name:  "MetricToPod",
		Start: classSpec{Domain: "nosuchdomain", Classes: []string{"x"}},
		Goal:  classSpec{Domain: "k8s", Classes: []string{"Pod"}},
	})
	require.NoError(t, err)
	assert.Nil(t, r)
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
		"require": func(v any) any { return rules.Require(v) },
	})
	r, err := rules.NewTemplateRule([]korrel8r.Class{start}, []korrel8r.Class{goal}, tmpl,
		`k8s:Pod:{"namespace":"{{or (index .Labels "k8s_namespace_name") (index .Labels "namespace") | require}}","name":"{{or (index .Labels "k8s_pod_name") (index .Labels "pod") | require}}"}`,
		d)
	require.NoError(b, err)
	return r
}
