// mock implementation of korrel8 interfaces for testing.
// Also serves as a handy template/reference when implementing a new  domain.
package mock

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
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

func (d Domain) URLRewriter(string) korrel8.URLRewriter { return nil }

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

func NewObjects(objectStrings ...string) []korrel8.Object {
	var ko []korrel8.Object
	for _, o := range objectStrings {
		ko = append(ko, Object(o))
	}
	return ko
}

var _ korrel8.Object = Object("") // Implements interface

type Rule struct {
	name        string
	start, goal korrel8.Class
	apply       func(korrel8.Object, *korrel8.Constraint) (*korrel8.Query, error)
}

func (r Rule) Start() korrel8.Class { return r.start }
func (r Rule) Goal() korrel8.Class  { return r.goal }
func (r Rule) String() string       { return r.name }
func (r Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (*korrel8.Query, error) {
	return r.apply(start, c)
}

var _ korrel8.Rule = Rule{} // Implements interface

func NewRule(name, start, goal string, apply func(korrel8.Object, *korrel8.Constraint) (*korrel8.Query, error)) Rule {
	return Rule{
		name:  name,
		start: Class(start),
		goal:  Class(goal),
		apply: apply,
	}
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

type Store struct{}

// Get treats the keys of q.Query as object strings, ignores the values
func (s Store) Get(_ context.Context, q *korrel8.Query, r korrel8.Result) error {
	for o := range q.Query() {
		r.Append(Object(o))
	}
	return nil
}

func NewQuery(objects ...Object) *korrel8.Query {
	v := url.Values{}
	for _, o := range objects {
		v[string(o)] = []string{""}
	}
	return &url.URL{RawQuery: v.Encode()}
}

var _ korrel8.Store = Store{} // Implements interface

// FIXME more intuitive, consistent mock strings.
