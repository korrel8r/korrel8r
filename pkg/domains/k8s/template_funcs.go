// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/rhobs/kube-health/pkg/analyze"
	khstatus "github.com/rhobs/kube-health/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TemplateFuncs for the k8s domain.
//
//	k8sClass
//		Takes string arguments (apiVersion, kind).
//		Returns the korrel8r.Class implied by the arguments, or an error.
//
//	k8sIsNamespaced
//		Takes a k8s Class argument, returns true if the class is a namespace-scoped resource.
//
//	k8sHealthStatus
//		Takes a k8s Object, evaluates its health using the kube-health library.
//		Returns "Error", "Warning", or "" for healthy/unknown objects.
//		Analyzes observed generation and standard Kubernetes conditions.
//
//	k8sCRDName
//	    Takes string arguments (apiVersion, kind).
//	    Returns the CustomResourceDefinition name (plural.group) for the resource,
//	    or "" if it is a core resource or unknown.
func (d *domain) TemplateFuncs() map[string]any {
	return map[string]any{
		"k8sClass": func(apiVersion, kind string) korrel8r.Class {
			return Class(schema.FromAPIVersionAndKind(apiVersion, kind))
		},
		"k8sIsNamespaced": func(c korrel8r.Class) bool {
			kc, ok := c.(Class)
			return ok && kc.Namespaced()
		},
		"k8sHealthStatus": HealthStatus,
		"k8sCRDName":      func(apiVersion, kind string) string { return d.CRDName(apiVersion, kind) },
	}
}

// CRDName returns the CustomResourceDefinition name (plural.group) for the resource
// defined ab kind/apiVersion, or returns "" if unknown.
func (d *domain) CRDName(apiVersion, kind string) string {
	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	if gvk.Group == "" {
		return "" // Core resources don't have CRDs.
	}
	c := Class(gvk)
	d.m.Lock()
	r := d.resources[c]
	d.m.Unlock()
	if r.Name == "" {
		return "" // Unknown resource.
	}
	return r.Name + "." + gvk.Group
}

// HealthStatus evaluates the health of a k8s object using the kube-health library.
// Returns "Error", "Warning", or "" for healthy/unknown objects.
func HealthStatus(o Object) string {
	if _, ok := o["status"]; !ok {
		return ""
	}
	obj, err := khstatus.NewObjectFromUnstructured(ToUnstructured(o))
	if err != nil {
		log.V(5).Info("k8sHealthStatus: failed to parse object", "error", err)
		return ""
	}
	conditions := analyze.AnalyzeObservedGeneration(obj)
	conds, err := analyze.AnalyzeObjectConditions(obj, analyze.DefaultConditionAnalyzers)
	if err != nil {
		log.V(5).Info("k8sHealthStatus: failed to analyze conditions", "error", err)
		return ""
	}
	conditions = append(conditions, conds...)
	result := analyze.AggregateResult(obj, nil, conditions)
	switch result.ObjStatus.Result {
	case khstatus.Error:
		return "Error"
	case khstatus.Warning:
		return "Warning"
	default:
		return ""
	}
}
