package logs

import "regexp"

func (_ domain) TemplateFuncs() map[string]any { return funcs }

var (
	funcs    = map[string]any{"lokiFixLabel": FixLabel}
	labelBad = regexp.MustCompile(`^[^a-zA-Z_:]|[^a-zA-Z0-9_:]`)
)

func FixLabel(label string) string { return labelBad.ReplaceAllString(label, "_") }
