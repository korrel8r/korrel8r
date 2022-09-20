// package templaterule implements korrel8.Rule as a Go template.
package templaterule

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/alanconway/korrel8/pkg/korrel8"
)

// Rule implements korrel8.Rule as a Go template that generate a query string from the start object.
// The template should return the empty string if the rule does not apply to the start object.
type Rule struct {
	*template.Template
	start, goal korrel8.Class
}

var funcs = map[string]any{
	"error": func(msg string) (struct{}, error) { return struct{}{}, errors.New(msg) },
}

func New(start, goal korrel8.Class, t *template.Template) *Rule {
	return &Rule{Template: t.Funcs(funcs), start: start, goal: goal}
}

func (r *Rule) String() string       { return fmt.Sprintf("%v(%v)->%v", r.Name(), r.start, r.goal) }
func (r *Rule) Start() korrel8.Class { return r.start }
func (r *Rule) Goal() korrel8.Class  { return r.goal }

// Follow the rule by applying the template to start.Native()
func (r *Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result korrel8.Queries, err error) {
	b := &strings.Builder{}
	data := start.Native()
	err = r.Execute(b, data)
	return []string{b.String()}, err
}

var _ korrel8.Rule = &Rule{}
