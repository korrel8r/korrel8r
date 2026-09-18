// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rules implements [korrel8r.Rule] and provides helper functions shared by rule implementations.
//
// Two rule types are provided:
//   - [NewTemplateRule]: rules defined by Go [text/template] strings, parsed from YAML configuration at runtime.
//   - [NewQuickRule]: rules using pre-compiled [quicktemplate] functions for type-safe, faster evaluation.
//
// Helper functions like [Empty], [Require], [Default], [Fail] and [ToJSON] are used by both
// rule types and by the sub-package [github.com/korrel8r/korrel8r/pkg/rules/quickrules].
//
// [quicktemplate]: https://github.com/valyala/quicktemplate
package rules

import (
	"bytes"
	"fmt"
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
// It takes a new, unparsed template and so allow Funcs and sub-templates can be included.
// It parses the query string into the unparsed template. Rule name is the template name.
func NewTemplateRule(start, goal []korrel8r.Class, unparsed *template.Template, query string, domains *korrel8r.Domains) (korrel8r.Rule, error) {
	r := &templateRule{start: start, goal: goal, query: unparsed, domains: domains}
	var err error
	r.query, err = r.query.Funcs(
		template.FuncMap{"currentRule": func() korrel8r.Rule { return r }},
	).Parse(query)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *templateRule) Name() string            { return r.query.Name() }
func (r *templateRule) String() string          { return r.Name() }
func (r *templateRule) Start() []korrel8r.Class { return r.start }
func (r *templateRule) Goal() []korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
//
// Returns (nil, err) if template execution returns a non-nil error.
// Returns (nil, nil) if template result is blank (all spaces)
func (r *templateRule) Apply(start korrel8r.Object) (queries []korrel8r.Query, err error) {
	defer func() {
		if p := recover(); p != nil {
			queries, err = nil, fmt.Errorf("%v", p)
		}
	}()
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	defer bufPool.Put(b)
	if err := r.query.Execute(b, start); err != nil {
		return nil, err
	}
	return parseQueries(r.domains, b.String())
}

// parseQueries converts a rule result string into a list of queries.
// The string may be blank (all whitespace) meaning the rule does not apply, or a list of
// query strings separated by newlines. Returns an error if any line is an invalid query.
// If a query implements [korrel8r.Expander], it is expanded into multiple queries.
func parseQueries(domains *korrel8r.Domains, result string) ([]korrel8r.Query, error) {
	var queries []korrel8r.Query
	for q := range strings.SplitSeq(result, "\n") {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		query, err := domains.Query(q)
		if err != nil {
			return nil, err
		}
		if expander, ok := query.(korrel8r.Expander); ok {
			if expanded := expander.Expand(); len(expanded) > 0 {
				queries = append(queries, expanded...)
				continue
			}
		}
		queries = append(queries, query)
	}
	return queries, nil
}
