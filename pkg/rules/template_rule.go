// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rules uses templates to generate goal queries from start objects.
package rules

import (
	"fmt"
	"text/template"

	"bytes"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// NewTemplateRule returns a korrel8r.Rule that uses Go templates to transform objects to queries.
//
// The funcsions in [Funcs] are available to the template.
func NewTemplateRule(start, goal korrel8r.Class, query *template.Template) korrel8r.Rule {
	return &templateRule{start: start, goal: goal, query: query}
}

type templateRule struct {
	query       *template.Template
	start, goal korrel8r.Class
}

func (r *templateRule) Name() string { return r.query.Name() }
func (r *templateRule) String() string {
	return fmt.Sprintf("%v(%v)->%v", r.Name(), r.Start().String(), r.Goal().String())
}
func (r *templateRule) Start() korrel8r.Class { return r.start }
func (r *templateRule) Goal() korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
func (r *templateRule) Apply(start korrel8r.Object) (korrel8r.Query, error) {
	b := &bytes.Buffer{}

	err := r.query.Funcs(map[string]any{
		"rule": func() korrel8r.Rule { return r },
	}).Execute(b, start)
	if err != nil {
		return nil, err
	}
	q, err := r.Goal().Domain().Query(b.String())
	if err != nil {
		return nil, err
	}
	if q.Data() == "" {
		return nil, fmt.Errorf("empty query")
	}
	if q.Class() != r.Goal() {
		return nil, fmt.Errorf("wrong goal %v, expected %v", q.Class(), r.Goal())
	}
	return q, nil
}
