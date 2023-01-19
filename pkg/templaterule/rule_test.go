package templaterule

import (
	"strings"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// rule makes a mock rule
func mockRule(name string, start, goal korrel8.Class) mock.Rule {
	return mock.NewRuleFromClasses(name, start, goal, nil)
}

// mockRules copies public parts of korrel8.Rule to a mock.Rule for easy comparison.
func mockRules(k []korrel8.Rule) []mock.Rule {
	m := make([]mock.Rule, len(k))
	for i := range k {
		m[i] = mock.NewRuleFromClasses(k[i].String(), k[i].Start(), k[i].Goal(), nil)
	}
	return m
}

func TestRule_Rules(t *testing.T) {
	e := engine.New()
	foo := mock.Domain("foo a b c")
	bar := mock.Domain("bar x y z")
	e.AddDomain(foo, nil)
	e.AddDomain(bar, nil)
	a, b, c := foo.Class("a"), foo.Class("b"), foo.Class("c")
	x, _, z := bar.Class("x"), bar.Class("y"), bar.Class("z")
	for _, x := range []struct {
		rule string
		want []mock.Rule
	}{
		{
			rule: `
name:   "simple"
start:  {domain: "foo", classes: [a]}
goal:   {domain: "bar", classes: [z]}
result: {uri: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule("simple", a, z)},
		},
		{
			rule: `
name: "multistart"
start: {domain: foo, classes: [a, b, c]}
goal:  {domain: bar, classes: [z]}
result: {uri: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule("multistart", a, z), mockRule("multistart", b, z), mockRule("multistart", c, z)},
		},
		{
			rule: `
name: "select-start"
start: {domain: foo, matches: ['{{assert (ne .Class.String "b")}}']}
goal:  {domain: bar, classes: [z]}
result: {uri: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule("select-start", a, z), mockRule("select-start", c, z)},
		},
		{
			rule: `
name: "select-goal"
start: {domain: foo, classes: [a]}
goal: {domain: bar, matches: ['{{assert (eq .Class.String "x")}}']}
result: {uri: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule("select-goal", a, x)},
		},
		{
			rule: `
name: "all-all"
start: {domain: foo}
goal: {domain: bar}
result: {uri: dummy, class: dummy}
`,
			want: func() []mock.Rule {
				var rules []mock.Rule
				for _, foo := range foo.Classes() {
					for _, bar := range bar.Classes() {
						rules = append(rules, mockRule("all-all", foo, bar))
					}
				}
				return rules
			}(),
		},
	} {
		t.Run(x.rule, func(t *testing.T) {
			var rule Rule
			require.NoError(t, yaml.Unmarshal([]byte(x.rule), &rule))
			got, err := rule.Rules(e)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.want, mockRules(got))
			}
		})
	}

	t.Run("implied-goal", func(t *testing.T) {
		r := Rule{
			Start:  ClassSpec{Domain: "foo", Classes: []string{"a"}},
			Goal:   ClassSpec{Domain: "bar", Classes: []string{"x"}},
			Result: ResultSpec{URI: "dummy"},
		}
		rs, err := r.Rules(e)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		b := &strings.Builder{}
		require.NoError(t, rs[0].(*rule).class.Execute(b, nil)) // Template should be a constant string.
		assert.Equal(t, "bar/x", b.String())
	})
}
