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

func addRule(name string, start korrel8.Class, body string) {
	t := template.Must(template.New(name).Parse(body))
	Rules = append(Rules, templaterule.New(start, Class{}, t))
}
