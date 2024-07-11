// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE
//
// # Template Functions
//
//	logTypeForNamespace
//	    Takes a namespace string argument.
//	    Returns the log type ("application" or "infrastructure") of a container in the namespace.
//
//	logSafeLabel
//	    Convert the string argument into a  safe label containing only alphanumerics '_' and ':'.
package log

import "regexp"

// TemplateFuncs for this domain. See package description.
func (domain) TemplateFuncs() map[string]any { return funcs } // TODO document template functions.

var (
	funcs = map[string]any{
		"logSafeLabel":        SafeLabel,
		"logTypeForNamespace": logTypeForNamespace,
	}
	labelBad = regexp.MustCompile(`^[^a-zA-Z_:]|[^a-zA-Z0-9_:]`)
)

// Returns a valid Loki stream label by replacing illegal characters in its argument with "_"
func SafeLabel(label string) string { return labelBad.ReplaceAllString(label, "_") }
func logTypeForNamespace(namespace string) string {
	if infraNamespace.MatchString(namespace) {
		return "infrastructure"
	}
	return "application"
}

var infraNamespace = regexp.MustCompile(`^(default|(openshift|kube)(-.*)?)$`)
