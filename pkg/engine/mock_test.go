package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
)

// Mock implementations

type mockDomain struct{}

func (d mockDomain) String() string                  { return "mock" }
func (d mockDomain) Class(name string) korrel8.Class { return mockClass(name) }
func (d mockDomain) KnownClasses() []korrel8.Class   { return nil }

var _ korrel8.Domain = mockDomain{} // Implements interface

type mockClass string

func (c mockClass) Domain() korrel8.Domain         { return mockDomain{} }
func (c mockClass) New() korrel8.Object            { return mockObject{} }
func (c mockClass) String() string                 { return string(c) }
func (c mockClass) Contains(o korrel8.Object) bool { _, ok := o.(*mockObject); return ok }
func (c mockClass) Key(o korrel8.Object) any       { return o }

var _ korrel8.Class = mockClass("") // Implements interface

type mockObject struct {
	name  string
	class mockClass
}

func o(name, class string) korrel8.Object { return mockObject{name: name, class: mockClass(class)} }

var _ korrel8.Object = mockObject{} // Implements interface

type mockRule struct {
	start, goal korrel8.Class
	apply       func(korrel8.Object, *korrel8.Constraint) korrel8.Query
}

func (r mockRule) Start() korrel8.Class { return r.start }
func (r mockRule) Goal() korrel8.Class  { return r.goal }
func (r mockRule) String() string       { return fmt.Sprintf("(%v)->%v", r.start, r.goal) }
func (r mockRule) Apply(start korrel8.Object, c *korrel8.Constraint) (korrel8.Query, error) {
	return r.apply(start, c), nil
}

var _ korrel8.Rule = mockRule{} // Implements interface

func rr(start, goal string, apply func(korrel8.Object, *korrel8.Constraint) korrel8.Query) mockRule {
	return mockRule{
		start: mockClass(start),
		goal:  mockClass(goal),
		apply: apply,
	}
}

func r(start, goal string) mockRule { return rr(start, goal, nil) }

type mockStore struct{}

// Query a mock "query" is a comma-separated list of "name.class" to be turned into mock objects.
func (s mockStore) Get(_ context.Context, q korrel8.Query, r korrel8.Result) error {
	for _, s := range strings.Split(string(q), ",") {
		nc := strings.Split(s, ".")
		r.Append(mockObject{name: nc[0], class: mockClass(nc[1])})
	}
	return nil
}

var _ korrel8.Store = mockStore{} // Implements interface
