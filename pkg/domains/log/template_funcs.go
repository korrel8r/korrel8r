// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import "regexp"

// TemplateFuncs for this domain. See package description.
func (domain) TemplateFuncs() map[string]any { return templateFuncs }

// See package documentation
var templateFuncs = map[string]any{
	"logSafeLabel":        SafeLabel,
	"logSafeLabels":       SafeLabels,
	"logTypeForNamespace": logTypeForNamespace,
}

var labelBad = regexp.MustCompile(`^[^a-zA-Z_:]|[^a-zA-Z0-9_:]`)

// Returns a valid Loki stream label by replacing illegal characters in its argument with "_"
func SafeLabel(label string) string { return labelBad.ReplaceAllString(label, "_") }

func SafeLabels(labels map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range labels {
		result[SafeLabel(k)] = v
	}
	return labels
}

func logTypeForNamespace(namespace string) string {
	if infraNamespace.MatchString(namespace) {
		return Infrastructure
	}
	return Application
}

var infraNamespace = regexp.MustCompile(`^(default|(openshift|kube)(-.*)?)$`)
