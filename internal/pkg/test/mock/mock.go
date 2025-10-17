// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mock is a mock implementation of a korrel8r domain for testing.
package mock

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	slices2 "github.com/korrel8r/korrel8r/pkg/slices"
)

var (
	// Validate implementation of interfaces.
	_ korrel8r.Domain = &Domain{}
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query{}
	_ korrel8r.Rule   = &Rule{}
	_ korrel8r.Store  = &Store{}
)

type Object any // mock.Object is any JSON-marshalable object.

type Domain struct {
	name    string
	classes []korrel8r.Class
}

func NewDomain(name string, classes ...string) *Domain {
	d := &Domain{name: name}
	for _, c := range classes {
		d.classes = append(d.classes, Class{name: c, domain: d})
	}
	return d
}
func (d *Domain) Name() string                          { return d.name }
func (d *Domain) String() string                        { return d.Name() }
func (d *Domain) Description() string                   { return "Mock domain." }
func (d *Domain) Store(cfg any) (korrel8r.Store, error) { return NewStoreConfig(d, cfg) }
func (d *Domain) Class(name string) korrel8r.Class {
	c := korrel8r.Class(Class{name: name, domain: d})
	if len(d.classes) > 0 && slices.Index(d.classes, c) < 0 {
		return nil
	}
	return c
}
func (d *Domain) Classes() []korrel8r.Class { return d.classes }

func (d *Domain) Query(query string) (korrel8r.Query, error) {
	domainName, className, selector := querySplit(query)
	if domainName != d.Name() {
		return nil, fmt.Errorf("wrong query domain, want %v: %v", d.Name(), query)
	}
	class := d.Class(className)
	if class == nil {
		return nil, fmt.Errorf("class not found: %v%v%v", domainName, korrel8r.NameSeparator, className)

	}
	return NewQuery(class, selector), nil
}

type Class struct {
	name   string
	domain korrel8r.Domain
}

func (c Class) Domain() korrel8r.Domain  { return c.domain }
func (c Class) String() string           { return classString(c) }
func (c Class) Name() string             { return c.name }
func (c Class) ID(o korrel8r.Object) any { return o }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) {
	var o Object
	err := json.Unmarshal(b, &o)
	return o, err
}

type ApplyFunc = func(korrel8r.Object) ([]korrel8r.Query, error)

type Rule struct {
	name        string
	start, goal []korrel8r.Class
	apply       ApplyFunc
}

// NewRule creates a rule: apply can be an [ApplyFunc], [korrel8r.Query] or nil.
func NewRule(name string, start, goal []korrel8r.Class, apply any) *Rule {
	r := &Rule{name: name, start: start, goal: goal}
	switch apply := apply.(type) {
	case ApplyFunc:
		r.apply = apply
	case korrel8r.Query:
		r.apply = func(korrel8r.Object) ([]korrel8r.Query, error) { return []korrel8r.Query{apply}, nil }
	case nil:
		r.apply = func(korrel8r.Object) ([]korrel8r.Query, error) {
			return nil, fmt.Errorf("mock rule has no result: %v", r)
		}
	default:
		panic(fmt.Errorf("expected korrel8r.Query or mock.ApplyFunc, got: (%T)%v", apply, apply))
	}
	return r
}

func (r *Rule) Start() []korrel8r.Class { return r.start }
func (r *Rule) Goal() []korrel8r.Class  { return r.goal }
func (r *Rule) Name() string            { return r.name }
func (r *Rule) String() string          { return r.Name() }

func (r *Rule) Apply(start korrel8r.Object) ([]korrel8r.Query, error) { return r.apply(start) }

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
	err    error
}

func NewQuery(c korrel8r.Class, selector string, result ...korrel8r.Object) korrel8r.Query {
	return Query{class: c, data: selector, result: result}
}

func NewQueryError(c korrel8r.Class, selector string, err error) korrel8r.Query {
	return Query{class: c, data: selector, err: err}
}

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { return q.data }
func (q Query) String() string        { return queryString(q) }

// Timestamper interface for objects with a Timestamp() method.
type Timestamper interface{ Timestamp() time.Time }

// Result implements Appender by appending to a slice.
type Result []korrel8r.Object

func (r *Result) Append(objects ...korrel8r.Object) { *r = append(*r, objects...) }
func (r Result) List() []korrel8r.Object            { return []korrel8r.Object(r) }

// Helper functions to replace impl dependencies
const sep = korrel8r.NameSeparator

func classString(c korrel8r.Class) string { return c.Domain().Name() + sep + c.Name() }

func queryString(q korrel8r.Query) string { return classString(q.Class()) + sep + q.Data() }

func querySplit(query string) (domain, class, data string) {
	query = strings.TrimSpace(query)
	s := strings.SplitN(query, sep, 3)
	if len(s) > 0 {
		domain = s[0]
	}
	if len(s) > 1 {
		class = s[1]
	}
	if len(s) > 2 {
		data = s[2]
	}
	return domain, class, data
}

// Builder provides convenience functions for creating mock queries and rules from string values.
type Builder struct {
	Domains []korrel8r.Domain
}

// NewBuilder -  domains can be a korrel8r.Domain or a string to create a mock domain.
func NewBuilder(domains ...any) *Builder {
	b := &Builder{}
	for _, d := range domains {
		switch d := d.(type) {
		case korrel8r.Domain:
			b.Domains = append(b.Domains, d)
		case string:
			b.Domains = append(b.Domains, NewDomain(d))
		default:
			panic(fmt.Errorf("expecting korrel8r.Domain or string, got: (%T)%v", d, d))
		}
	}
	return b
}

func (b *Builder) Domain(name string) korrel8r.Domain {
	i := slices.IndexFunc(b.Domains, func(d korrel8r.Domain) bool { return name == d.Name() })
	if i < 0 {
		panic(fmt.Errorf("mock builder: unknown domain: %v", name))
	}
	return b.Domains[i]
}

func (b *Builder) Class(v any) korrel8r.Class {
	switch v := v.(type) {
	case korrel8r.Class:
		return v
	case string:
		d, c, _ := strings.Cut(v, korrel8r.NameSeparator)
		return b.Domain(d).Class(c)
	default:
		panic(fmt.Errorf("mock builder: unknown class: (%T)%v", v, v))
	}
}

func (b *Builder) Classes(vs ...any) (classes []korrel8r.Class) {
	for _, v := range vs {
		switch v := v.(type) {
		case []korrel8r.Class:
			classes = append(classes, v...)
		case []any:
			classes = append(classes, b.Classes(v...)...)
		case []string:
			classes = append(classes, b.Classes(slices2.Anys(v)...)...)
		default:
			classes = append(classes, b.Class(v))
		}
	}
	return classes
}

// Rule creates a new rule. start and goal can be a single class or class name, or a slice.
// See [NewRule] for parameter details.
func (b *Builder) Rule(name string, start, goal, apply any) korrel8r.Rule {
	return NewRule(name, b.Classes(start), b.Classes(goal), apply)
}

// result can be [error] or [object...]
func (b *Builder) Query(class any, selector string, result ...any) korrel8r.Query {
	if len(result) == 1 { // Possible error
		if err, ok := result[0].(error); ok {
			return NewQueryError(b.Class(class), selector, err)
		}
	}
	return NewQuery(b.Class(class), selector, result...)
}

func (b *Builder) Store(domain string, cfg any) korrel8r.Store {
	s, err := b.Domain(domain).Store(cfg)
	if err != nil {
		panic(fmt.Errorf("mock builder: store error: %w", err))
	}
	return s
}
