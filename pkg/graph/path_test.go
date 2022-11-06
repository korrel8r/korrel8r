package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/stretchr/testify/assert"
)

func links(start, goal string, rules ...string) Links {
	var e Links
	if len(rules) == 0 {
		rules = []string{fmt.Sprintf("%v:%v", start, goal)}
	}
	for _, r := range rules {
		e = append(e, mock.NewRule(r, start, goal, nil))
	}
	return e
}

func TestLinks_String(t *testing.T) {
	assert.Equal(t, "[]", fmt.Sprint(Links{}))
	assert.Equal(t, "[]", fmt.Sprint(Links(nil)))
	assert.Equal(t, "[x y z]", fmt.Sprint(links("a", "b", "x", "y", "z")))
}

func TestMultiPath_String(t *testing.T) {
	assert.Equal(t, "[]", fmt.Sprint(MultiPath{}))
	assert.Equal(t, "[]", fmt.Sprint(MultiPath(nil)))
	assert.Equal(t, "[(a)[x y z](b)[u v](c)]", fmt.Sprint(MultiPath{links("a", "b", "x", "y", "z"), links("b", "c", "u", "v")}))
}
