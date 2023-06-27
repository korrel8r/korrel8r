// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

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
func NewTemplateRule(start, goal korrel8r.Class, query, constraint *template.Template) korrel8r.Rule {
	return &templateRule{start: start, goal: goal, query: query, constraint: constraint}
}

type templateRule struct {
	query, constraint *template.Template
	start, goal       korrel8r.Class
}

func (r *templateRule) String() string        { return r.query.Name() }
func (r *templateRule) Start() korrel8r.Class { return r.start }
func (r *templateRule) Goal() korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
func (r *templateRule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (korrel8r.Query, error) {
	b := &bytes.Buffer{}

	err := r.query.Funcs(map[string]any{
		"constraint": func() *korrel8r.Constraint { return c },
		"rule":       func() korrel8r.Rule { return r },
	}).Execute(b, start)
	if err != nil {
		return nil, fmt.Errorf("apply: %s", err)
	}

	q, err := r.Goal().Domain().Query(b.String())
	if err != nil {
		return nil, fmt.Errorf("apply: unmarshal error: %w", err)
	}

	if q.Class() != r.Goal() {
		return nil, fmt.Errorf("apply: wrong goal: %v != %v; in %#+v",
			korrel8r.ClassName(r.Goal()), korrel8r.ClassName(q.Class()), q)
	}

	return q, nil
}
