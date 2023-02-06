// package mock is a mock implementation of a korrel8r domain for testing.
//
// Useful as a starting point when implementing a new domain.
package mock

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var (
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain("")
	_ korrel8r.Class  = Class("")
	_ korrel8r.Query  = Query("")
	_ korrel8r.Rule   = Rule{}
	_ korrel8r.Store  = Store{}
)

var NoMockErr = errors.New("mock not implemented")

// Domain implemented as a string.
//
// NOTE a korrel8r.Domain has just a single instance.
// In most cases you would define a type called `domain`,  with a single instance variable called `Domain`
type Domain string

func (d Domain) String() string {
	if f := strings.Fields(string(d)); len(f) > 0 {
		return f[0]
	} else {
		return string(d)
	}
}

func (d Domain) Class(name string) korrel8r.Class {
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

func (d Domain) Classes() (classes []korrel8r.Class) {
	for _, c := range strings.Fields(string(d))[1:] {
		classes = append(classes, d.Class(c))
	}
	return classes
}

func (Domain) UnmarshalQuery(b []byte) (korrel8r.Query, error) { panic(NoMockErr) }

// Query implemented as a string
type Query string

func (q Query) String() string        { return string(q) }
func (q Query) Class() korrel8r.Class { panic(NoMockErr) }

// Class implemented as a string of the form "domain/class"
type Class string

func (c Class) Domain() korrel8r.Domain {
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

func (c Class) New() korrel8r.Object     { return Object(fmt.Sprintf("%v:", string(c))) }
func (c Class) ID(o korrel8r.Object) any { return o }

// Object string is "class:data"
type Object string

func (o Object) Class() Class { return Class(strings.Split(string(o), ":")[0]) }
func (o Object) Data() string { return strings.Split(string(o), ":")[1] }

func Objects(objectStrings ...string) []korrel8r.Object {
	var ko []korrel8r.Object
	for _, o := range objectStrings {
		ko = append(ko, Object(o))
	}
	return ko
}

type ApplyFunc = func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error)

type Rule struct {
	name        string
	start, goal korrel8r.Class
	apply       ApplyFunc
}

func (r Rule) Start() korrel8r.Class { return r.start }
func (r Rule) Goal() korrel8r.Class  { return r.goal }
func (r Rule) String() string        { return r.name }
func (r Rule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (korrel8r.Query, error) {
	return r.apply(start, c)
}

func NewRule(name, start, goal string, apply ApplyFunc) Rule {
	return NewRuleFromClasses(name, Class(start), Class(goal), apply)
}

func NewRuleFromClasses(name string, start, goal korrel8r.Class, apply ApplyFunc) Rule {
	return Rule{name: name, start: start, goal: goal, apply: apply}
}

func QuickRule(start, goal string) Rule { return NewRule(start+":"+goal, start, goal, nil) }

// Rules creates QuickRule rules from start, goal pairs, apply is nil.
func Rules(startGoal ...string) []korrel8r.Rule {
	var rules []korrel8r.Rule
	for i := 0; i < len(startGoal); i += 2 {
		rules = append(rules, QuickRule(startGoal[i], startGoal[i+1]))
	}
	return rules
}

// Store is a map of mock query strings to sets of objects.
type Store map[string][]korrel8r.Object

func (s Store) Domain() korrel8r.Domain { panic(NoMockErr) }

// Get returns the objects associated with the query
func (s Store) Get(_ context.Context, q korrel8r.Query, r korrel8r.Appender) error {
	mq := q.(Query)
	for _, o := range s[mq.String()] {
		r.Append(o)
	}
	return nil
}

func (s Store) Resolve(korrel8r.Query) *url.URL { panic("not implemented") }

// NewQuery returns a query that will return the given objects.
func (s Store) NewQuery(objs ...string) korrel8r.Query {
	r := Query(strings.Join(objs, "&"))
	s[r.String()] = Objects(objs...)
	return r
}
