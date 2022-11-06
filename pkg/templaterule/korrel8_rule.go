package templaterule

import (
	"net/url"
	"text/template"

	"bytes"

	"github.com/korrel8/korrel8/pkg/korrel8"
)

// rule implements korrel8.rule as a Go template that generate a query string from the start object.
type rule struct {
	*template.Template
	start, goal korrel8.Class
}

func (r *rule) String() string       { return r.Template.Name() }
func (r *rule) Start() korrel8.Class { return r.start }
func (r *rule) Goal() korrel8.Class  { return r.goal }

// Apply the rule by applying the template.
// The template will be executed with start as the "." context object.
// A function "constraint" returns the constraint.
func (r *rule) Apply(start korrel8.Object, c *korrel8.Constraint) (*korrel8.Query, error) {
	b := &bytes.Buffer{}
	err := r.Template.Funcs(map[string]any{"constraint": func() *korrel8.Constraint { return c }}).Execute(b, start)
	if err != nil {
		return nil, err
	}
	return url.Parse(b.String())
}

var _ korrel8.Rule = &rule{}
