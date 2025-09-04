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
		return nil, korrel8r.ClassNotFoundError(className)
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

type ApplyFunc func(korrel8r.Object) (korrel8r.Query, error)

type Rule struct {
	name        string
	start, goal []korrel8r.Class
	apply       ApplyFunc
}

// NewRule creates a rule: apply can be an [ApplyFunc] or a [korrel8r.Query].
func NewRule(name string, start, goal []korrel8r.Class, apply any) *Rule {
	switch apply := apply.(type) {
	case func(korrel8r.Object) (korrel8r.Query, error):
		return NewRuleFunc(name, start, goal, apply)
	case korrel8r.Query:
		return NewRuleQuery(name, start, goal, apply)
	default:
		panic(fmt.Errorf("expected korrel8r.Query or mock.ApplyFunc, got: (%T)%v", apply, apply))
	}
}

// NewRuleQuery create rule, [Apply] calls apply.
func NewRuleFunc(name string, start, goal []korrel8r.Class, apply ApplyFunc) *Rule {
	return &Rule{name: name, start: start, goal: goal, apply: apply}
}

// NewRuleQuery create rule, [Apply] returns a fixed query.
func NewRuleQuery(name string, start, goal []korrel8r.Class, q korrel8r.Query) *Rule {
	apply := func(korrel8r.Object) (korrel8r.Query, error) { return q, nil }
	return NewRuleFunc(name, start, goal, apply)
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
