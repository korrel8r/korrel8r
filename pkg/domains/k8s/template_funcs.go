// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TemplateFuncs for the k8s domain
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

func k8sClass(apiVersion, kind string) Class {
	return Class(schema.FromAPIVersionAndKind(apiVersion, kind))
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
