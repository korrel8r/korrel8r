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
	prep        func(korrel8.Object) any
}

func New(start, goal korrel8.Class, t *template.Template) *Rule {
	return &Rule{Template: t, start: start, goal: goal}
}

func (r *Rule) String() string       { return fmt.Sprintf("%v(%v)->%v", r.Name(), r.start, r.goal) }
func (r *Rule) Start() korrel8.Class { return r.start }
func (r *Rule) Goal() korrel8.Class  { return r.goal }

// TemplateData interface provides the template starting data if different from the object itself.
type TemplateData interface{ TemplateData() any }

// Follow the rule by applying the template. If start implements TemplateData, use that as the template input.
func (r *Rule) Follow(start korrel8.Object) (result korrel8.Result, err error) {
	if r.Start() != start.Class() {
		return result, fmt.Errorf("rule %v wants %v got %v", r, r.Start(), start.Class())
	}
	b := &strings.Builder{}
	var data any = start
	if td, ok := start.(TemplateData); ok {
		data = td.TemplateData()
	}
	r.Execute(b, data)
	return korrel8.Result{Domain: r.Goal().Domain(), Queries: []string{b.String()}}, nil
}

var _ korrel8.Rule = &Rule{}
