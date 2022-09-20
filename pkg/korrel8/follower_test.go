package korrel8

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFollower_Follow(t *testing.T) {
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
			r, err := mockFollower.Follow(context.Background(), o("foo", "a"), nil, x.path)
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, r)
		})
	}
}

func TestFollower_FollowEach(t *testing.T) {
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
			r, err := mockFollower.followEach(x.rule, x.start, nil)
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, r)
		})
	}
}
