package templaterule

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// rule makes a mock rule
func mockRule(name string, start, goal korrel8r.Class) mock.Rule {
	return mock.NewRuleFromClasses(name, start, goal, nil)
}

// mockRules copies public parts of korrel8r.Rule to a mock.Rule for easy comparison.
func mockRules(k ...korrel8r.Rule) []mock.Rule {
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
	_, _, z := bar.Class("x"), bar.Class("y"), bar.Class("z")
	for _, x := range []struct {
		rule string
		want []mock.Rule
	}{
		{
			rule: `
name:   "simple"
start:  {domain: "foo", classes: [a]}
goal:   {domain: "bar", classes: [z]}
result: {query: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule("simple", a, z)},
		},
		{
			rule: `
name: "multistart"
start: {domain: foo, classes: [a, b, c]}
goal:  {domain: bar, classes: [z]}
result: {query: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule("multistart", a, z), mockRule("multistart", b, z), mockRule("multistart", c, z)},
		},
		{
			rule: `
name: "all-all"
start: {domain: foo}
goal: {domain: bar}
result: {query: dummy, class: dummy}
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
				assert.ElementsMatch(t, x.want, mockRules(got...))
			}
		})
	}
}
