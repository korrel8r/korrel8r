package korrel8impl

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/alanconway/korrel8/pkg/korrel8"
)

// TemplateRule implements korrel8.Rule with a Go template that generate a query string from the start object.
type TemplateRule struct {
	t           *template.Template
	start, goal korrel8.Class
	prep        func(korrel8.Object) any
}

// TemplateData interface provides the template starting data if different from the object itself.
type TemplateData interface{ TemplateData() any }

func NewTemplateRule(name string, start, goal korrel8.Class, body string) (*TemplateRule, error) {
	r := &TemplateRule{t: template.New(name), start: start, goal: goal}
	_, err := r.t.Parse(body)
	return r, err
}

func (r TemplateRule) String() string       { return fmt.Sprintf("%v(%v)->%v", r.t.Name(), r.start, r.goal) }
func (r TemplateRule) Start() korrel8.Class { return r.start }
func (r TemplateRule) Goal() korrel8.Class  { return r.goal }

// Follow the rule by applying the template. If start implements TemplateData, use that as the template input.
func (r TemplateRule) Follow(start korrel8.Object) (result korrel8.Result, err error) {
	if r.Start() != start.Class() {
		return result, fmt.Errorf("rule %v wants %v got %v", r, r.Start(), start.Class())
	}
	b := &strings.Builder{}
	var data any = start
	if td, ok := start.(TemplateData); ok {
		data = td.TemplateData()
	}
	r.t.Execute(b, data)
	return korrel8.Result{Domain: r.Goal().Domain(), Queries: []string{b.String()}}, nil
}

var _ korrel8.Rule = &TemplateRule{}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
