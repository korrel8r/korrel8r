// package templaterule implements korrel8.Rule as a Go template.
package templaterule

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/alanconway/korrel8/pkg/korrel8"
)

// Rule implements korrel8.Rule as a Go template that generate a query string from the start object.
type Rule struct {
	*template.Template
	start, goal korrel8.Class
}

func New(start, goal korrel8.Class, t *template.Template) *Rule {
	return &Rule{Template: t, start: start, goal: goal}
}

func (r *Rule) String() string       { return fmt.Sprintf("%v(%v)->%v", r.Name(), r.start, r.goal) }
func (r *Rule) Start() korrel8.Class { return r.start }
func (r *Rule) Goal() korrel8.Class  { return r.goal }

// Follow the rule by applying the template to start.Native()
func (r *Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result korrel8.Queries, err error) {
	b := &strings.Builder{}
	var data any = start
	data = start.Native()
	r.Execute(b, data)
	return []string{b.String()}, nil
}

var _ korrel8.Rule = &Rule{}
