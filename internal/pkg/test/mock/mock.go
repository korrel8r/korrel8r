// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mock is a mock implementation of a korrel8r domain for testing.
package mock

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain("")
	_ korrel8r.Class  = Domain("").Class("")
	_ korrel8r.Query  = Query{}
	_ korrel8r.Rule   = &Rule{}
	_ korrel8r.Store  = &Store{}
)

type Object any // mock.Object is any JSON-marshalable object.

type Domain string

func (d Domain) Name() string                          { return string(d) }
func (d Domain) String() string                        { return d.Name() }
func (d Domain) Description() string                   { return "Mock domain." }
func (d Domain) Class(name string) korrel8r.Class      { return Class{name: name, domain: d} }
func (d Domain) Classes() (classes []korrel8r.Class)   { return nil }
func (d Domain) Store(cfg any) (korrel8r.Store, error) { return NewStoreConfig(d, cfg) }

func Classes(d korrel8r.Domain, names ...string) []korrel8r.Class {
	c := make([]korrel8r.Class, len(names))
	for i, n := range names {
		c[i] = d.Class(n)
	}
	return c
}

func (d Domain) Query(s string) (korrel8r.Query, error) {
	class, data, err := impl.ParseQuery(d, s)
	return NewQuery(class, data), err
}

func Domains(names ...string) []korrel8r.Domain {
	var domains []korrel8r.Domain
	for _, name := range names {
		domains = append(domains, Domain(name))
	}
	return domains
}

type DomainWithClasses struct {
	Domain
	MClasses []korrel8r.Class
}

func NewDomainWithClasses(name string, classes ...string) *DomainWithClasses {
	d := &DomainWithClasses{Domain: Domain(name)}
	for _, c := range classes {
		d.MClasses = append(d.MClasses, Class{name: c, domain: d})
	}
	return d
}

func (d DomainWithClasses) Class(name string) korrel8r.Class {
	i := slices.IndexFunc(d.MClasses, func(c korrel8r.Class) bool { return c.Name() == name })
	if i < 0 {
		return nil
	}
	return d.MClasses[i]
}

func (d DomainWithClasses) Classes() []korrel8r.Class { return d.MClasses }

type Class struct {
	name   string
	domain korrel8r.Domain
}

func (c Class) Domain() korrel8r.Domain                     { return c.domain }
func (c Class) String() string                              { return impl.ClassString(c) }
func (c Class) Name() string                                { return c.name }
func (c Class) Description() string                         { return fmt.Sprintf("mock class %v", c.String()) }
func (c Class) ID(o korrel8r.Object) any                    { return o }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }

type ApplyFunc func(korrel8r.Object) (korrel8r.Query, error)

type Rule struct {
	name        string
	start, goal []korrel8r.Class
	apply       ApplyFunc
}

// NewRule constructs a rule, result must be one of:
//
//	Korrel8r.Query
//	func(korrel8r.Object) (korrel8r.Query, error)
//	mock.Store, result-values...
func NewRule(name string, start, goal []korrel8r.Class, result ...any) *Rule {
	r := &Rule{name: name, start: start, goal: goal}
	var ok bool
	if r.apply, ok = result[0].(func(korrel8r.Object) (korrel8r.Query, error)); ok {
		return r // reslut is the apply function
	}
	q, ok := result[0].(korrel8r.Query) // Try for a fixed query
	if !ok {                            // Try for a store/class fixed result query
		s, c := result[0].(*Store), result[1].(korrel8r.Class)
		q = s.NewQuery(c, result[2:]...)
	}
	r.apply = func(korrel8r.Object) (korrel8r.Query, error) { return q, nil }
	return r
}

func (r *Rule) Start() []korrel8r.Class { return r.start }
func (r *Rule) Goal() []korrel8r.Class  { return r.goal }
func (r *Rule) Name() string            { return r.name }
func (r *Rule) String() string          { return r.Name() }

func (r *Rule) Apply(start korrel8r.Object) (korrel8r.Query, error) { return r.apply(start) }

// RuleLess orders rules.
func RuleLess(a, b korrel8r.Rule) int {
	if a.Start()[0].Name() != b.Start()[0].Name() {
		return strings.Compare(a.Start()[0].Name(), b.Start()[0].Name())
	}
	return strings.Compare(a.Goal()[0].Name(), b.Goal()[0].Name())
}

// SorRules  sorts rules by (start, goal) order.
func SortRules(rules []korrel8r.Rule) []korrel8r.Rule { slices.SortFunc(rules, RuleLess); return rules }

// Query implements korrel8r.Query
type Query struct {
	class  korrel8r.Class
	data   string
	result []korrel8r.Object
}

func NewQuery(c korrel8r.Class, selector string, result ...korrel8r.Object) korrel8r.Query {
	return Query{class: c, data: selector, result: result}
}

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { return q.data }
func (q Query) String() string        { return impl.QueryString(q) }

// Timestamper interface for objects with a Timestamp() method.
type Timestamper interface{ Timestamp() time.Time }

// ListResult implements Appender by appending to a slice.
type ListResult []korrel8r.Object

func (r *ListResult) Append(objects ...korrel8r.Object) { *r = append(*r, objects...) }
func (r ListResult) List() []korrel8r.Object            { return []korrel8r.Object(r) }
