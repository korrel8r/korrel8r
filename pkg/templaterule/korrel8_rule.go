// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package templaterule

import (
	"fmt"
	"text/template"

	"bytes"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var _ korrel8r.Rule = &rule{}

// rule implements korrel8r.Rule
type rule struct {
	query, constraint *template.Template
	start, goal       korrel8r.Class
}

func (r *rule) String() string        { return r.query.Name() }
func (r *rule) Start() korrel8r.Class { return r.start }
func (r *rule) Goal() korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
// The template will be executed with start as the "." context object.
// A function "constraint" returns the constraint.
func (r *rule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (korrel8r.Query, error) {
	b := &bytes.Buffer{}

	err := r.query.Funcs(map[string]any{
		"constraint": func() *korrel8r.Constraint { return c },
		"rule":       func() korrel8r.Rule { return r },
	}).Execute(b, start)
	if err != nil {
		return nil, fmt.Errorf("apply: %s", err)
	}

	q, err := r.Goal().Domain().UnmarshalQuery(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("apply: unmarshal error: %w", err)
	}

	if q.Class() != r.Goal() {
		return nil, fmt.Errorf("apply: wrong goal: %v != %v; in %#+v",
			korrel8r.ClassName(r.Goal()), korrel8r.ClassName(q.Class()), q)
	}

	return q, nil
}
