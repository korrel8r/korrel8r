package korrel8

import (
	"fmt"
	"strings"
	"text/template"
)

// TemplateRule uses a Go template to generate a query string from the context object.
type TemplateRule struct {
	t             *template.Template
	context, goal Class
}

func NewTemplateRule(name string, context, goal Class, body string) (*TemplateRule, error) {
	r := &TemplateRule{t: template.New(name), context: context, goal: goal}
	_, err := r.t.Parse(body)
	return r, err
}

func (r TemplateRule) String() string { return r.t.Name() }
func (r TemplateRule) Context() Class { return r.context }
func (r TemplateRule) Goal() Class    { return r.goal }

func (r TemplateRule) Follow(context any) (Query, error) {
	if !r.Context().Contains(context) {
		return "", fmt.Errorf("following rule %v: context value (%T)%v not in expected class %v ", r, context, context, r.Context())
	}
	b := &strings.Builder{}
	r.t.Execute(b, context)
	return Query(b.String()), nil
}
