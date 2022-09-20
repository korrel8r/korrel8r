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

func AddTo(r *korrel8.RuleSet) {
	r.Add(K8sToK8s()...)
	r.Add(AlertToK8s()...)
	r.Add(K8sToLoki()...)
}

func All() *korrel8.RuleSet { rs := korrel8.NewRuleSet(); AddTo(rs); return rs }
