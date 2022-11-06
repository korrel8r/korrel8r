package templaterule

import (
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func init() {
	logging.Init(0)
}

// Edges turns a slice of rules into a slice of [[startClassName, endClassName]...]
func edges(rules []korrel8.Rule) (edges [][]korrel8.Class) {
	for _, r := range rules {
		edges = append(edges, []korrel8.Class{r.Start(), r.Goal()})
	}
	return edges
}

func TestRule_Rules(t *testing.T) {
	e := engine.New("")
	foo := mock.Domain("foo a b c")
	bar := mock.Domain("bar x y z")
	e.AddDomain(foo, nil)
	e.AddDomain(bar, nil)
	a, b, c := foo.Class("a"), foo.Class("b"), foo.Class("c")
	x, y, z := bar.Class("x"), bar.Class("y"), bar.Class("z")

	for _, x := range []struct {
		rule string
		want [][]korrel8.Class // [[start1,goal1], [start2,goal2]...]
	}{
		{
			rule: `{name: simple, start: [foo, a], goal: [bar, z], query: 'bar/z:hello'}`,
			want: [][]korrel8.Class{{a, z}},
		},
		{
			rule: `{name: multi-start, start: [foo, a, b, c], goal: [bar, z], query: 'bar/z:hello'}`,
			want: [][]korrel8.Class{{a, z}, {b, z}, {c, z}},
		},
		{
			rule: `{name: "template start", start: [foo, '{{ne .Class.String "b"}}'], goal: [bar, z], query: 'bar/z:hello'}`,
			want: [][]korrel8.Class{{a, z}, {c, z}},
		}, {
			rule: `{name: "template goal", start: [foo, a], goal: [bar, '{{.Class.String}}:hello'], query: 'bar/z:hello'}`,
			want: [][]korrel8.Class{{a, x}, {a, y}, {a, z}},
		},
	} {
		t.Run(x.rule, func(t *testing.T) {
			rule := Rule{}
			require.NoError(t, yaml.Unmarshal([]byte(x.rule), &rule))
			got, err := rule.Rules(e)
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, edges(got))
			}
		})
	}

	// 	emptyClass := mock.ParseClass("")
	// 	tr, err := templaterule.New("myrule", emptyClass, emptyClass, `/mock?name={{.Name}}&constraint={{constraint}}`)
	// 	assert.NoError(t, err)
	// 	now := time.Now()
	// 	constraint := korrel8.Constraint{Start: &now, End: &now}
	// 	q, err := tr.Apply(mock.NewObject("thing", ""), &constraint)
	// 	assert.NoError(t, err)
	// 	assert.Equal(t, fmt.Sprintf("/mock?name=thing&constraint=%v", constraint), q.String())
	// }

	// func TestRule_Error(t *testing.T) {
	// 	tr, err := templaterule.New("myrule", mock.Class(""), mock.Class(""), `{{fail "foobar"}}`)
	// 	assert.NoError(t, err)
	// 	_, err = tr.Apply(mock.NewObject("thing", ""), nil)
	// 	want := "template: myrule:1:2: executing \"myrule\" at <fail \"foobar\">: error calling fail: foobar"
	// 	assert.EqualError(t, err, want)
	// }

	// func TestRule_MissingKey(t *testing.T) {
	// 	tr, err := templaterule.New(t.Name(), mock.Class(""), mock.Class(""), `{{.nosuchkey}}`)
	// 	assert.NoError(t, err)
	// 	_, err = tr.Apply(mock.NewObject("thing", ""), nil)
	// 	assert.Contains(t, err.Error(), "can't evaluate field nosuchkey")
}
