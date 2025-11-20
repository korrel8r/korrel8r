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
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (d *domain) TemplateFuncs() map[string]any {
	return map[string]any{
		"k8sClass": func(apiVersion, kind string) korrel8r.Class {
			return Class(schema.FromAPIVersionAndKind(apiVersion, kind))
		},
		"k8sIsNamespaced": func(c korrel8r.Class) bool {
			kc, ok := c.(Class)
			return ok && kc.Namespaced()
		},
	}
}

// k8sClass returns a k8s.Class from an apiVersion and kind string.
