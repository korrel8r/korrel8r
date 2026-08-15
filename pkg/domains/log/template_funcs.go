// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"fmt"
	"reflect"
	"regexp"
)

// TemplateFuncs for the log domain.
//
//	logTypeForNamespace string
//	    Returns the log type ("application" or "infrastructure") for logs from that namespace.
//
//	logSafeLabel string
//	    Replace all characters other than alphanumerics, '_' and ':' with '_'.
//
//	logSafeLabels map[string]string
//	    Returns a map where each key is replaced by the result of logSafeLabel.
//
// [LogQL]: https://grafana.com/docs/loki/latest/query
func (domain) TemplateFuncs() map[string]any {
	return map[string]any{
		"logSafeLabel":        SafeLabel,
		"logSafeLabels":       SafeLabels,
		"logTypeForNamespace": TypeForNamespace,
	}
}

var labelBad = regexp.MustCompile(`^[^a-zA-Z_:]|[^a-zA-Z0-9_:]`)

// Returns a valid Loki stream label by replacing illegal characters in its argument with "_"
func SafeLabel(label string) string { return labelBad.ReplaceAllString(label, "_") }

func SafeLabels(labelMap any) (any, error) {
	in := reflect.ValueOf(labelMap)
	if in.Kind() != reflect.Map || in.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("safeLabels: expecting map[string]T, got %T", labelMap)
	}
	out := reflect.MakeMap(in.Type())
	i := in.MapRange()
	for i.Next() {
		k := SafeLabel(i.Key().String())
		out.SetMapIndex(reflect.ValueOf(k), i.Value())
	}
	return out.Interface(), nil
}

// TypeForNamespace returns the log type ("application" or "infrastructure") for the given namespace.
func TypeForNamespace(namespace string) string {
	if infraNamespace.MatchString(namespace) {
		return Infrastructure.Name()
	}
	return Application.Name()
}

var infraNamespace = regexp.MustCompile(`^(default|(openshift|kube)(-.*)?)$`)
