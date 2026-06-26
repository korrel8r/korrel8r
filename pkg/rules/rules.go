// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rules uses Go templates to generate goal queries from start objects.
//
// See [github.com/korrel8r/korrel8r/pkg/config.Rule] for details of configuring a rule.
package rules

import (
	"bytes"
	"strings"
	"sync"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

var _ korrel8r.Rule = &templateRule{}

type templateRule struct {
	query       *template.Template
	start, goal []korrel8r.Class
	domains     *korrel8r.Domains
}

// NewTemplateRule returns a korrel8r.Rule that uses a Go template to transform objects to queries.
// The domains registry is used to parse generated query strings.
func NewTemplateRule(start, goal []korrel8r.Class, query *template.Template, domains *korrel8r.Domains) korrel8r.Rule {
	return &templateRule{start: start, goal: goal, query: query, domains: domains}
}

func (r *templateRule) Name() string            { return r.query.Name() }
func (r *templateRule) String() string          { return r.Name() }
func (r *templateRule) Start() []korrel8r.Class { return r.start }
func (r *templateRule) Goal() []korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
//
// Returns (nil, err) if template execution returns a non-nil error.
// Returns (nil, nil) if template result is blank (all spaces)
func (r *templateRule) Apply(start korrel8r.Object) ([]korrel8r.Query, error) {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	defer bufPool.Put(b)
	if err := r.query.Execute(b, start); err != nil {
		return nil, err
	}
	var queries []korrel8r.Query
	for q := range strings.SplitSeq(b.String(), "\n") {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		query, err := r.domains.Query(q)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}
	return queries, nil
}
