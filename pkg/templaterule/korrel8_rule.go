package templaterule

import (
	"text/template"

	"bytes"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
)

// rule implements korrel8.Rule
type rule struct {
	uri, class, constraint *template.Template
	start, goal            korrel8.Class
}

func (r *rule) String() string       { return r.uri.Name() }
func (r *rule) Start() korrel8.Class { return r.start }
func (r *rule) Goal() korrel8.Class  { return r.goal }

// Apply the rule by applying the template.
// The template will be executed with start as the "." context object.
// A function "constraint" returns the constraint.
func (r *rule) Apply(start korrel8.Object, c *korrel8.Constraint) (uri.Reference, error) {
	b := &bytes.Buffer{}
	if err := r.uri.Funcs(map[string]any{"constraint": func() *korrel8.Constraint { return c }}).Execute(b, start); err != nil {
		return uri.Reference{}, err
	}
	return uri.Parse(b.String())
}

var _ korrel8.Rule = &rule{}
