package rules

import (
	"text/template"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/templaterule"
)

func newTemplate(name string, start, goal korrel8.Class, tmpl string) korrel8.Rule {
	t := template.Must(template.New(name).Parse(tmpl))
	return templaterule.New(start, goal, t)
}

var FuncMap = map[string]any{}

func Rules() []korrel8.Rule {
	return append(K8sToK8s(), K8sToLoki()...)
}
