package loki

import (
	"text/template"

	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/templaterule"
	v1 "k8s.io/api/core/v1"
)

var Rules []korrel8.Rule

func init() {
	addRule("PodLogs", k8s.ClassOf(&v1.Pod{}),
		`{kubernetes_namespace_name="{{.ObjectMeta.Namespace}}",kubernetes_pod_name="{{.ObjectMeta.Name}}"}`)
}

type rule struct{ *templaterule.Rule }

func (r rule) Follow(start korrel8.Object, c *korrel8.Constraint) (result korrel8.Result, err error) {
	result, err = r.Rule.Follow(start, c)
	for i, q := range result {
		result[i] = addConstraint(q, c)
	}
	return result, err
}

func addConstraint(q string, c *korrel8.Constraint) string {
	if c == nil {
		return q
	}
	return QueryObject{
		Query: q,
		Start: c.After,
		End:   c.Before,
	}.String()
}

// FIXME need test for constraints

func addRule(name string, start korrel8.Class, body string) {
	t := template.Must(template.New(name).Parse(body))
	Rules = append(Rules, rule{templaterule.New(start, Class{}, t)})
}
