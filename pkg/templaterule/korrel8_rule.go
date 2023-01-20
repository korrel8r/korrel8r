package templaterule

import (
	"text/template"

	"bytes"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/uri"
)

// rule implements korrel8r.Rule
type rule struct {
	uri, class, constraint *template.Template
	start, goal            korrel8r.Class
}

func (r *rule) String() string       { return r.uri.Name() }
func (r *rule) Start() korrel8r.Class { return r.start }
func (r *rule) Goal() korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
// The template will be executed with start as the "." context object.
// A function "constraint" returns the constraint.
func (r *rule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (uri.Reference, error) {
	b := &bytes.Buffer{}
	if err := r.uri.Funcs(map[string]any{"constraint": func() *korrel8r.Constraint { return c }}).Execute(b, start); err != nil {
		return uri.Reference{}, err
	}
	return uri.Parse(b.String())
}

var _ korrel8r.Rule = &rule{}
