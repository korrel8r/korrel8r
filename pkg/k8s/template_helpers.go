package k8s

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (s *Store) TemplateHelpers() map[string]any {
	return map[string]any{
		"k8sResource": func(kind, apiVersion string) (string, error) {
			return kindToResource(s.c.RESTMapper(), kind, apiVersion)
		},
		"k8sClass": kindToClass,
	}
}

// kindToResource convert a kind and apiVersion to a resource string.
func kindToResource(restMapper meta.RESTMapper, kind string, apiVersion string) (resource string, err error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return "", err
	}
	gk := schema.GroupKind{Group: gv.Group, Kind: kind}
	rm, err := restMapper.RESTMapping(gk, gv.Version)
	if err != nil {
		return "", err
	}
	return rm.Resource.Resource, nil
}

func kindToClass(kind, apiVersion string) (string, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return "", err
	}
	return Class(gv.WithKind(kind)).String(), nil
}
