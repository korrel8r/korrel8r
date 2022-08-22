package k8s

import (
	"github.com/alanconway/korrel8/pkg/korrel8"
	corev1 "k8s.io/api/core/v1"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustV[T any](v T, err error) T {
	must(err)
	return v
}

var (
	ServiceClass = mustV(ClassOf(&corev1.Service{}))
	PodClass     = mustV(ClassOf(&corev1.Pod{}))

	// FIXME working example rules.
	ServicePodsRule = mustV(korrel8.NewTemplateRule("ServicePods", ServiceClass, PodClass, `
	{{/* FIXME template to produce selector query from service spec */}}
`))
)
