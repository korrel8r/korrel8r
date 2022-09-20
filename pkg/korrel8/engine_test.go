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
	for i, x := range []struct {
		path Path
		want Queries
	}{
		{
			path: Path{
				// Return 2 results, must follow both
				rr("a", "b", func(Object, *Constraint) Queries { return []string{"1.b", "2.b"} }),
				// Return the start object's name with the goal class.
				rr("b", "c", func(start Object, _ *Constraint) Queries { return []string{start.(mockObject).name + ".c", "x.c"} }),
				rr("c", "z", func(start Object, _ *Constraint) Queries { return []string{start.(mockObject).name + ".z", "y.z"} }),
			},
			want: Queries{"1.z", "2.z", "x.z", "y.z"},
		},
		{
			path: nil,
			want: Queries{},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			e := NewEngine()
			e.Add(mockDomain{}, mockStore{})
			r, err := e.Follow(context.Background(), o("foo", "a"), nil, x.path)
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, r)
		})
	}
}

func TestEngine_FollowEach(t *testing.T) {
	for i, x := range []struct {
		rule  Rule
		start []Object
		want  Queries
	}{
		{
			rule:  rr("a", "b", func(start Object, c *Constraint) Queries { return []string{start.(mockObject).name + ".b", "x.b"} }),
			start: []Object{o("1", "a"), o("2", ".a"), o("1", "a")},
			want:  Queries{"1.b", "2.b", "x.b"},
		},
		{
			rule: r("a", "b"),
			want: Queries{},
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
