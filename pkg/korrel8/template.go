package korrel8

import (
	"fmt"
	"strings"
	"text/template"
)

// TemplateRule uses a Go template to generate a query string from the start object.
type TemplateRule struct {
	t           *template.Template
	start, goal Class
}

func NewTemplateRule(name string, start, goal Class, body string) (*TemplateRule, error) {
	r := &TemplateRule{t: template.New(name), start: start, goal: goal}
	_, err := r.t.Parse(body)
	return r, err
}

func MustNewTemplateRule(name string, start, goal Class, body string) *TemplateRule {
	return must(NewTemplateRule(name, start, goal, body))
}

func (r TemplateRule) String() string { return r.t.Name() }
func (r TemplateRule) Start() Class   { return r.start }
func (r TemplateRule) Goal() Class    { return r.goal }

func (r TemplateRule) Follow(start Object) (Query, error) {
	if !r.Start().Contains(start) {
		return "", fmt.Errorf("following rule %v(%v) got %T", r, r.Start(), start)
	}
	b := &strings.Builder{}
	r.t.Execute(b, start)
	return Query(b.String()), nil
}

var _ Rule = &TemplateRule{}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
