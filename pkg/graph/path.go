package graph

import (
	"fmt"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/slices"
)

// Links are a set of rules with the same Start() and Goal().
// They form one step on a MultiPath.
type Links []korrel8r.Rule

func (e Links) Start() korrel8r.Class { return safeApply(e, 0, korrel8r.Rule.Start) }
func (e Links) Goal() korrel8r.Class  { return safeApply(e, 0, korrel8r.Rule.Goal) }
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

// Sort into consistent order for comparison
func (l Links) Sort() {
	slices.SortFunc(l, func(a, b korrel8r.Rule) bool { return a.String() < b.String() })
}

// MultiPath represents multiple paths from a Start to a Goal.
type MultiPath []Links

func (path MultiPath) Start() korrel8r.Class {
	if len(path) > 0 {
		return path[0].Start()
	}
	return nil
}

func (path MultiPath) Goal() korrel8r.Class {
	if len(path) > 0 {
		return path[len(path)-1].Goal()
	}
	return nil
}

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

// Sort link lists into consistent order for comparison of MultiPaths.
func (mp MultiPath) Sort() {
	for _, links := range mp {
		links.Sort()
	}
}

func (mp MultiPath) String() string {
	if len(mp) == 0 {
		return "[]"
	}
	b := &strings.Builder{}
	b.WriteString(("["))
	for _, links := range mp {
		fmt.Fprintf(b, "%v %v ", links.Start(), links)
	}
	fmt.Fprintf(b, "%v", mp[len(mp)-1].Goal())
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
