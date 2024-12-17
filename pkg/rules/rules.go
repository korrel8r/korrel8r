// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rules uses Go templates to generate goal queries from start objects.
//
// See [github.com/korrel8r/korrel8r/pkg/config.Rule] for details of configuring a rule.
package rules

import (
	"errors"
	"strings"
	"text/template"

	"bytes"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

// NewTemplateRule returns a korrel8r.Rule that uses a Go template to transform objects to queries.
func NewTemplateRule(start, goal []korrel8r.Class, query *template.Template) korrel8r.Rule {
	return &templateRule{start: start, goal: goal, query: query}
}

var _ = impl.AssertRule(&templateRule{})

type templateRule struct {
	query       *template.Template
	start, goal []korrel8r.Class
}

func (r *templateRule) Name() string            { return r.query.Name() }
func (r *templateRule) String() string          { return r.Name() }
func (r *templateRule) Start() []korrel8r.Class { return r.start }
func (r *templateRule) Goal() []korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
// Return non-nil error if the rule does not apply.
func (r *templateRule) Apply(start korrel8r.Object) (korrel8r.Query, error) {
	b := &bytes.Buffer{}
	if err := r.query.Execute(b, start); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(string(b.String()))
	if query == "" { // Blank query means rule does not apply.
		return nil, errors.New("No query generated")
	}
	return r.Goal()[0].Domain().Query(query)
}
