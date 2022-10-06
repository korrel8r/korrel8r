package korrel8

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Parse(t *testing.T) {
	for _, x := range []struct {
		name string
		want Class
		err  string
	}{
		{"mock/foo", mockDomain{}.Class("foo"), ""},
		{"mock", mockDomain{}.Class(""), ""}, // Allow "default" class, empty string
		{"nosuch", nil, `unknown domain: "nosuch"`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := NewEngine()
			e.Add(mockDomain{}, nil)
			c, err := e.ParseClass(x.name)
			if x.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, x.err)
			}
			assert.Equal(t, x.want, c)
		})
	}
}

func TestEngine_Follow(t *testing.T) {
	path := Path{
		// Return 2 results, must follow both
		rr("a", "b", func(Object, *Constraint) []string { return []string{"1.b", "2.b"} }),
		// Replace start object's class with goal class
		rr("b", "c", func(start Object, _ *Constraint) []string { return []string{start.(mockObject).name + ".c", "x.c"} }),
		rr("c", "z", func(start Object, _ *Constraint) []string { return []string{start.(mockObject).name + ".z", "y.z"} }),
	}
	want := []string{"1.z", "2.z", "x.z", "y.z"}

	e := NewEngine()
	e.Add(mockDomain{}, mockStore{})
	queries, err := e.Follow(context.Background(), o("foo", "a"), nil, path)
	assert.NoError(t, err)
	assert.ElementsMatch(t, want, queries)
}

func TestEngine_FollowEach(t *testing.T) {
	for i, x := range []struct {
		rule  Rule
		start []Object
		want  []string
	}{
		{
			rule:  rr("a", "b", func(start Object, c *Constraint) []string { return []string{start.(mockObject).name + ".b", "x.b"} }),
			start: []Object{o("1", "a"), o("2", ".a"), o("1", "a")},
			want:  []string{"1.b", "2.b", "x.b"},
		},
		{
			rule: r("a", "b"),
			want: []string{},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			e := NewEngine()
			e.Add(mockDomain{}, mockStore{})
			r, err := e.followEach(x.rule, x.start, nil)
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, r)
		})
	}
}
