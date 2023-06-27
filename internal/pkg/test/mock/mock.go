// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mock is a mock implementation of a korrel8r domain for testing.
// All the mock types (Domain, Class, Object etc.) are simply integers that implement the korrel8r interfaces.
// This simplifies tests since they can be initialized from int constants.
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/slices"
)

var (
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain("")
	_ korrel8r.Class  = Domain("").Class("")
	_ korrel8r.Query  = Query{}
	_ korrel8r.Rule   = Rule{}
	_ korrel8r.Store  = NewStore(Domain(""))
)

type Domain string

func (d Domain) String() string                                     { return string(d) }
func (d Domain) Class(name string) korrel8r.Class                   { return Class{name: name, domain: d} }
func (d Domain) Classes() (classes []korrel8r.Class)                { return nil }
func (d Domain) Store(korrel8r.StoreConfig) (korrel8r.Store, error) { return NewStore(d), nil }
func (d Domain) Query(s string) (korrel8r.Query, error) {
	q := &query{}
	err := json.Unmarshal([]byte(s), q)
	return NewQuery(d.Class(q.Class), q.Results...), err
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
	i := slices.IndexFunc(d.MClasses, func(c korrel8r.Class) bool { return c.String() == name })
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

func (c Class) Domain() korrel8r.Domain  { return c.domain }
func (c Class) String() string           { return c.name }
func (c Class) ID(o korrel8r.Object) any { return o }
func (c Class) New() korrel8r.Object     { return "" }

type Rule struct {
	name        string
	start, goal korrel8r.Class
}

func NewRule(name string, start, goal korrel8r.Class) Rule {
	return Rule{name: name, start: start, goal: goal}
}

func NewRules(rules ...korrel8r.Rule) (mocks []Rule) {
	for _, r := range rules {
		mocks = append(mocks, NewRule(r.String(), r.Start(), r.Goal()))
	}
	return mocks
}

func (r Rule) Start() korrel8r.Class { return r.start }
func (r Rule) Goal() korrel8r.Class  { return r.goal }
func (r Rule) String() string        { return fmt.Sprintf("%v->%v", r.start, r.goal) }
func (r Rule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (korrel8r.Query, error) {
	panic("not implemented") // See ApplyRule
}

// RuleLess orders rules.
func RuleLess(a, b korrel8r.Rule) bool {
	if a.Start().String() != b.Start().String() {
		return a.Start().String() < b.Start().String()
	}
	return a.Goal().String() < b.Goal().String()
}

// SorRules  sorts rules by (start, goal) order.
func SortRules(rules []korrel8r.Rule) []korrel8r.Rule { slices.SortFunc(rules, RuleLess); return rules }

type ApplyFunc func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error)
type ApplyRule struct {
	Rule
	apply ApplyFunc
}

func NewApplyRule(start, goal korrel8r.Class, apply ApplyFunc) ApplyRule {
	return ApplyRule{Rule: Rule{start: start, goal: goal}, apply: apply}
}

func NewQueryRule(start korrel8r.Class, query korrel8r.Query) ApplyRule {
	return NewApplyRule(start, query.Class(), func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error) {
		return query, nil
	})
}
func (r ApplyRule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (korrel8r.Query, error) {
	return r.apply(start, c)
}

// Store is a mock store, use with [Query]
type Store struct{ domain korrel8r.Domain }

func NewStore(d korrel8r.Domain) Store { return Store{domain: d} }

func (s Store) Domain() korrel8r.Domain { return s.domain }
func (s Store) Get(_ context.Context, q korrel8r.Query, r korrel8r.Appender) error {
	r.Append(q.(Query).MResults...)
	return nil
}

func (s *Store) Resolve(korrel8r.Query) *url.URL { panic("not implemented") }

// Query is a mock query that contains the desired results.
type Query struct {
	MClass   korrel8r.Class
	MResults []korrel8r.Object
}

func NewQuery(c korrel8r.Class, results ...korrel8r.Object) korrel8r.Query {
	return Query{MClass: c, MResults: results}
}
func (q Query) Class() korrel8r.Class { return q.MClass }
func (q Query) String() string {
	b, _ := json.Marshal(query{q.MClass.String(), q.MResults})
	return string(b)
}

// query for marshal/unmarhsal
type query struct {
	Class   string            `json:"class"`
	Results []korrel8r.Object `json:"results"`
}
