package graph

import (
	"fmt"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
)

// Links are a set of rules with the same Start() and Goal().
// They form one step on a MultiPath.
type Links []korrel8.Rule

func (e Links) Start() korrel8.Class { return safeApply(e, 0, korrel8.Rule.Start) }
func (e Links) Goal() korrel8.Class  { return safeApply(e, 0, korrel8.Rule.Goal) }
func (l Links) Valid() bool {
	if len(l) > 1 {
		for _, r := range l[1:] {
			if r.Goal() != l[0].Goal() || r.Start() != l[0].Start() {
				return false
			}
		}
	}
	return true
}

// MultiPath represents multiple paths from a Start to a Goal.
type MultiPath []Links

func (path MultiPath) Valid() bool {
	for i, links := range path {
		if !links.Valid() {
			return false
		}
		if i < len(path)-1 && links.Goal() != path[i+1].Start() {
			return false
		}
	}
	return true
}

func (mp MultiPath) String() string {
	if len(mp) == 0 {
		return "[]"
	}
	b := &strings.Builder{}
	b.WriteString(("["))
	for _, links := range mp {
		fmt.Fprintf(b, "<%v> %v ", links.Start(), links)
	}
	fmt.Fprintf(b, "<%v>", mp[len(mp)-1].Goal())
	b.WriteString(("]"))
	return b.String()
}

func safeApply[T any, R any, S ~[]T](s S, i int, f func(T) R) R {
	if i < len(s) {
		return f(s[i])
	}
	return zero[R]()
}

func zero[T any]() T { var t T; return t }
