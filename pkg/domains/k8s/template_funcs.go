// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// # Template Functions
//
// The following template functions are available to rules.
//
//	k8sClass
//	    Takes string arguments (apiVersion, kind).
//	    Returns the korrel8r.Class implied by the arguments, or an error.
package k8s

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TemplateFuncs for this domain. See package description.
func (domain) TemplateFuncs() map[string]any {
	return map[string]any{
		"k8sClass": k8sClass,
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

func k8sClass(apiVersion, kind string) Class {
	return Class(schema.FromAPIVersionAndKind(apiVersion, kind))
}
