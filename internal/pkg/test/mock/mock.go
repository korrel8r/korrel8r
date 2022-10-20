// mock implementation of korrel8 interfaces for testing
package mock

import (
	"context"
	"fmt"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
)

type Domain struct{}

func (d Domain) String() string                  { return "mock" }
func (d Domain) Class(name string) korrel8.Class { return Class(name) }
func (d Domain) KnownClasses() []korrel8.Class   { return nil }

var _ korrel8.Domain = Domain{} // Implements interface

type Class string

func (c Class) Domain() korrel8.Domain         { return Domain{} }
func (c Class) New() korrel8.Object            { return Object{} }
func (c Class) String() string                 { return string(c) }
func (c Class) Contains(o korrel8.Object) bool { _, ok := o.(*Object); return ok }
func (c Class) Key(o korrel8.Object) any       { return o }

var _ korrel8.Class = Class("") // Implements interface

type Object struct {
	Name  string
	Class Class
}

func NewObject(name, class string) korrel8.Object { return Object{Name: name, Class: Class(class)} }

var _ korrel8.Object = Object{} // Implements interface

type Rule struct {
	start, goal korrel8.Class
	apply       func(korrel8.Object, *korrel8.Constraint) korrel8.Query
}

func (r Rule) Start() korrel8.Class { return r.start }
func (r Rule) Goal() korrel8.Class  { return r.goal }
func (r Rule) String() string       { return fmt.Sprintf("(%v)=%v", r.start, r.goal) }
func (r Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (korrel8.Query, error) {
	return r.apply(start, c), nil
}

var _ korrel8.Rule = Rule{} // Implements interface

func NewRule(start, goal string, apply func(korrel8.Object, *korrel8.Constraint) korrel8.Query) Rule {
	return Rule{
		start: Class(start),
		goal:  Class(goal),
		apply: apply,
	}
}

// Rules creates rules from start, goal pairs.
func NewRules(startGoal ...string) []korrel8.Rule {
	if len(startGoal)%2 != 0 {
		panic("NewRules needs an even number of arguments.")
	}
	var rules []korrel8.Rule
	for i := 0; i < len(startGoal); i += 2 {
		rules = append(rules, NewRule(startGoal[i], startGoal[i+1], nil))
	}
	return rules
}

type Store struct{}

// Query a  "query" is a comma-separated list of "name.class" to be turned into  objects.
func (s Store) Get(_ context.Context, q korrel8.Query, r korrel8.Result) error {
	for _, s := range strings.Split(string(q), ",") {
		nc := strings.Split(s, ".")
		r.Append(Object{Name: nc[0], Class: Class(nc[1])})
	}
	return nil
}

var _ korrel8.Store = Store{} // Implements interface
