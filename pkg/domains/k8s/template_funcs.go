// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TODO document

func (s *Store) TemplateFuncs() map[string]any {
	return map[string]any{
		"k8sMetricLabelKind": s.metricLabelKind,
		"k8sResource": func(kind, apiVersion string) (string, error) {
			return kindToResource(s.c.RESTMapper(), kind, apiVersion)
		}}
}

func (domain) TemplateFuncs() map[string]any {
	return map[string]any{
		"k8sClass":        k8sClass,
		"k8sQueryClass":   k8sQueryClass,
		"k8sGroupVersion": schema.ParseGroupVersion,
		"k8sLogType":      logType,
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

func k8sClass(kind, apiVersion string) (Class, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return Class{}, err
	}
	return Class(gv.WithKind(kind)), nil
}

func k8sQueryClass(classOrName any) (string, error) {
	var c Class
	switch classOrName := classOrName.(type) {
	case Class:
		c = classOrName
	case string:
		c = Domain.Class(classOrName).(Class)
	default:
		return "", fmt.Errorf("not a k8s class: %v", classOrName)
	}
	return fmt.Sprintf(`"Group": %q, "Version": %q, "Kind": %q`, c.Group, c.Version, c.Kind), nil
}

var infraNamespace = regexp.MustCompile(`^(default|(openshift|kube)(-.*)?)$`)

// logType returns the type (application or infrastructure) of a container log based on the namespace.
func logType(namespace string) string {
	if infraNamespace.MatchString(namespace) {
		return "infrastructure"
	}
	return "application"
}

// metricLabelKind attempt to match a metric label to a k8s Kind name.
func (s *Store) metricLabelKind(label string) *schema.GroupVersionKind {
	for _, gv := range s.groups {
		for kind := range Scheme.KnownTypes(gv) {
			if matchLabel(label, kind) {
				gvk := gv.WithKind(kind)
				return &gvk
			}
		}
	}
	return nil
}

func matchLabel(label, kind string) bool {
	label, kind = strings.ToLower(label), strings.ToLower(kind)
	return label == kind ||
		strings.TrimSuffix(label, "_name") == kind ||
		strings.TrimSuffix(label, "name") == kind
}
