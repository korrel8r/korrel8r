// mock implementation of korrel8 interfaces for testing.
// Also serves as a handy template/reference when implementing a new  domain.
package mock

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
)

// Domain string is "domain" or "domain class1 class2 ..." if Classes is needed.
type Domain string

func (d Domain) String() string {
	if f := strings.Fields(string(d)); len(f) > 0 {
		return f[0]
	} else {
		return string(d)
	}
}

func (d Domain) Class(name string) korrel8.Class {
	c := Class(fmt.Sprintf("%v/%v", d, name))
	if !strings.Contains(string(d), " ") {
		return c // no Classes, allow anything
	}
	// If there are Classes make sure this is one.
	if ok, _ := regexp.MatchString(fmt.Sprintf(` %v( |$)`, name), string(d)); ok {
		return c
	}
	return nil
}

func (d Domain) Classes() (classes []korrel8.Class) {
	for _, c := range strings.Fields(string(d))[1:] {
		classes = append(classes, d.Class(c))
	}
	return classes
}

var _ korrel8.Domain = Domain("") // Implements interface

// Class string is domain/class
type Class string

func (c Class) Domain() korrel8.Domain {
	if x, _, ok := strings.Cut(string(c), "/"); ok {
		return Domain(x)
	}
	return Domain("")
}

func (c Class) String() string {
	if x, y, ok := strings.Cut(string(c), "/"); ok {
		return y
	} else {
		return x
	}
}

func (c Class) New() korrel8.Object      { return Object(fmt.Sprintf("%v:", string(c))) }
func (c Class) Key(o korrel8.Object) any { return o }

var _ korrel8.Class = Class("") // Implements interface

// Object string is "class:data"
type Object string

func (o Object) Class() Class { return Class(strings.Split(string(o), ":")[0]) }
func (o Object) Data() string { return strings.Split(string(o), ":")[1] }

func Objects(objectStrings ...string) []korrel8.Object {
	var ko []korrel8.Object
	for _, o := range objectStrings {
		ko = append(ko, Object(o))
	}
	return ko
}

var _ korrel8.Object = Object("") // Implements interface

type ApplyFunc = func(korrel8.Object, *korrel8.Constraint) (uri.Reference, error)

type Rule struct {
	name        string
	start, goal korrel8.Class
	apply       ApplyFunc
}

func (r Rule) Start() korrel8.Class { return r.start }
func (r Rule) Goal() korrel8.Class  { return r.goal }
func (r Rule) String() string       { return r.name }
func (r Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (uri.Reference, error) {
	return r.apply(start, c)
}

var _ korrel8.Rule = Rule{} // Implements interface

func NewRule(name, start, goal string, apply ApplyFunc) Rule {
	return NewRuleFromClasses(name, Class(start), Class(goal), apply)
}

func NewRuleFromClasses(name string, start, goal korrel8.Class, apply ApplyFunc) Rule {
	return Rule{name: name, start: start, goal: goal, apply: apply}
}

func QuickRule(start, goal string) Rule { return NewRule(start+":"+goal, start, goal, nil) }

// Rules creates QuickRule rules from start, goal pairs, apply is nil.
func Rules(startGoal ...string) []korrel8.Rule {
	var rules []korrel8.Rule
	for i := 0; i < len(startGoal); i += 2 {
		rules = append(rules, QuickRule(startGoal[i], startGoal[i+1]))
	}
	return rules
}

// Store is a map of query URI strings to sets of objects.
type Store map[string][]korrel8.Object

// Get returns the objects associated with the query
func (s Store) Get(_ context.Context, ref uri.Reference, r korrel8.Result) error {
	for _, o := range s[ref.String()] {
		r.Append(o)
	}
	return nil
}

// NewReference returns a query that will return the given objects.
func (s Store) NewReference(objs ...string) uri.Reference {
	r := uri.Reference{Path: strings.Join(objs, "&")}
	s[r.String()] = Objects(objs...)
	return r
}

var _ korrel8.Store = Store{} // Implements interface
