// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import "regexp"

// TempalateFuncs returns template functions specific to this domain.
func (domain) TemplateFuncs() map[string]any { return funcs } // TODO document template functions.

var (
	funcs    = map[string]any{"lokiFixLabel": FixLabel}
	labelBad = regexp.MustCompile(`^[^a-zA-Z_:]|[^a-zA-Z0-9_:]`)
)

// Returns a valid Loki stream label by replacing illegal characters in its argument with "_"
func FixLabel(label string) string { return labelBad.ReplaceAllString(label, "_") }
