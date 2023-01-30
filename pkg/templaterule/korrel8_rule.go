package templaterule

import (
	"encoding/json"
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
func (r *rule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (q korrel8r.Query, err error) {
	b := &bytes.Buffer{}
	err = r.query.Funcs(map[string]any{"constraint": func() *korrel8r.Constraint { return c }}).Execute(b, start)
	if err != nil {
		return nil, err
	}
	q = r.Goal().Domain().Query(r.Goal())
	if err = json.Unmarshal(b.Bytes(), &q); err != nil {
		return nil, fmt.Errorf("%w: %v", err, b)
	}
	return q, err
}
