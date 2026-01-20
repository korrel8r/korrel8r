// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"fmt"
	"reflect"
	"regexp"
)

func (domain) TemplateFuncs() map[string]any {
	return map[string]any{
		"logSafeLabel":        SafeLabel,
		"logSafeLabels":       SafeLabels,
		"logTypeForNamespace": logTypeForNamespace,
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

func logTypeForNamespace(namespace string) string {
	if infraNamespace.MatchString(namespace) {
		return Infrastructure.Name()
	}
	return Application.Name()
}

var infraNamespace = regexp.MustCompile(`^(default|(openshift|kube)(-.*)?)$`)
