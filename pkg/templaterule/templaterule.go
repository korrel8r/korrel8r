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

// Funcs that are automatically added to templates created by New.
// Rule.Apply() also adds a "constraint" function.
var funcs = map[string]any{
	// doesnotapply fails template evaluation, call when a rule does not apply to its start object.
	"doesnotapply": func() (int, error) { return 0, errors.New("rule does not apply") },
	// constraint is a placeholder ,in Rule.Apply it will return a *Constraint (possibly nil)
	"constraint": func() *korrel8.Constraint { panic("placeholder") },
}

// New rule using a template to convert the start object to a goal query.
func New(name string, start, goal korrel8.Class, tmpl string) (*Rule, error) {
	t, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(tmpl)
	return &Rule{Template: t, start: start, goal: goal}, err
}

func (r *Rule) String() string       { return fmt.Sprintf("%v(%v)->%v", r.Template.Name(), r.start, r.goal) }
func (r *Rule) Start() korrel8.Class { return r.start }
func (r *Rule) Goal() korrel8.Class  { return r.goal }

// Follow the rule by applying the template to start.Native()
// The template will be executed with start as the "." context object and a function "constarint" that returns the constraint.
func (r *Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result []string, err error) {
	b := &strings.Builder{}
	data := start.Native()
	err = r.Template.Funcs(map[string]any{"constraint": func() *korrel8.Constraint { return c }}).Execute(b, data)
	return []string{string(b.String())}, err
}

var _ korrel8.Rule = &Rule{}
