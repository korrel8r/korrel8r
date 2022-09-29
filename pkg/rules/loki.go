package rules

import (
	"fmt"

	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/loki"
	"github.com/alanconway/korrel8/pkg/templaterule"
	v1 "k8s.io/api/core/v1"
)

// FIXME generalize applying constraints to rules, constructing rules.

type rule struct{ korrel8.Rule }

func (r rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result korrel8.Queries, err error) {
	result, err = r.Rule.Apply(start, c)
	return result, err
}

func (r rule) String() string { return fmt.Sprintf("%v", r.Rule) }

// FIXME need test for constraints

func K8sToLoki() []korrel8.Rule {
	return []korrel8.Rule{
		must(templaterule.New("PodToLokiLogs", k8s.ClassOf(&v1.Pod{}), loki.Class{},
			`query_range?direction=forward&query=
{{- urlquery (printf "{kubernetes_namespace_name=%q,kubernetes_pod_name=%q}" .ObjectMeta.Namespace .ObjectMeta.Name) }}
{{- with constraint}}&start={{constraint.Start.UnixNano}}{{end -}}
{{- with constraint}}&end={{constraint.End.UnixNano}}{{end -}}
`))}
	// FIXME reusalbe &param={quoted x=y list from map}
}
