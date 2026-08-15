// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules

import (
	"regexp"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
)

var (
	semverREStr  = `\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?`
	csvVersionRE = regexp.MustCompile(`\.v` + semverREStr + "$")
)

func csvPartOfName(csvName string) string { return csvVersionRE.ReplaceAllString(csvName, "") }

func logTypeForNamespace(ns string) string { return log.TypeForNamespace(ns) }

// isCustomResource returns true if the object's API group indicates a custom resource.
// Built-in groups are empty (core), have no dots (apps, batch), or end with .k8s.io.
func isCustomResource(o k8s.Object) bool {
	apiVersion, _ := o["apiVersion"].(string)
	group, _, _ := strings.Cut(apiVersion, "/")
	if group == apiVersion {
		return false // core group, e.g. "v1"
	}
	return strings.Contains(group, ".") && !strings.HasSuffix(group, ".k8s.io")
}

func k8sMetadata(o any) (obj k8s.Object, metadata map[string]any, ns, name, kind string) {
	obj = o.(k8s.Object)
	metadata, _ = obj["metadata"].(map[string]any)
	ns, _ = metadata["namespace"].(string)
	name, _ = metadata["name"].(string)
	kind, _ = obj["kind"].(string)
	return
}

func ingressServiceNames(obj map[string]any) []string {
	spec := obj["spec"].(map[string]any)
	var names []string
	if db, ok := spec["defaultBackend"].(map[string]any); ok {
		if svc, ok := db["service"].(map[string]any); ok {
			names = append(names, svc["name"].(string))
		}
	}
	for _, r := range mapSlice(spec, "rules") {
		rule := r.(map[string]any)
		httpRule, ok := rule["http"].(map[string]any)
		if !ok {
			continue
		}
		for _, p := range mapSlice(httpRule, "paths") {
			path := p.(map[string]any)
			if backend, ok := path["backend"].(map[string]any); ok {
				if svc, ok := backend["service"].(map[string]any); ok {
					names = append(names, svc["name"].(string))
				}
			}
		}
	}
	return names
}

// kvSecretNames extracts Secret names from kubevirt volumes and accessCredentials.
func kvSecretNames(volumes, creds []any) []string {
	var names []string
	for _, v := range volumes {
		vol := v.(map[string]any)
		if s, ok := vol["secret"].(map[string]any); ok {
			names = append(names, s["secretName"].(string))
		}
		for _, ciKey := range []string{"cloudInitNoCloud", "cloudInitConfigDrive"} {
			if ci, ok := vol[ciKey].(map[string]any); ok {
				if ref, ok := ci["secretRef"].(map[string]any); ok {
					names = append(names, ref["name"].(string))
				}
				if ref, ok := ci["networkDataSecretRef"].(map[string]any); ok {
					names = append(names, ref["name"].(string))
				}
			}
		}
		if sp, ok := vol["sysprep"].(map[string]any); ok {
			if s, ok := sp["secret"].(map[string]any); ok {
				names = append(names, s["name"].(string))
			}
		}
	}
	for _, c := range creds {
		cred := c.(map[string]any)
		for _, credKey := range []string{"sshPublicKey", "userPassword"} {
			if entry, ok := cred[credKey].(map[string]any); ok {
				if src, ok := entry["source"].(map[string]any); ok {
					if s, ok := src["secret"].(map[string]any); ok {
						names = append(names, s["secretName"].(string))
					}
				}
			}
		}
	}
	return names
}

// kvConfigMapNames extracts ConfigMap names from kubevirt volumes.
func kvConfigMapNames(volumes []any) []string {
	var names []string
	for _, v := range volumes {
		vol := v.(map[string]any)
		if cm, ok := vol["configMap"].(map[string]any); ok {
			names = append(names, cm["name"].(string))
		}
		if sp, ok := vol["sysprep"].(map[string]any); ok {
			if cm, ok := sp["configMap"].(map[string]any); ok {
				names = append(names, cm["name"].(string))
			}
		}
	}
	return names
}

func mapSlice(m map[string]any, key string) []any {
	switch v := m[key].(type) {
	case []any:
		return v
	case []map[string]any:
		r := make([]any, len(v))
		for i := range v {
			r[i] = v[i]
		}
		return r
	default:
		return nil
	}
}
