package templaterule

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"sigs.k8s.io/yaml"
)

// Funcs that are automatically added to templates created by New.
// Rule.Apply() adds a "constraint" function with the constraint if present.
var funcs = map[string]any{
	"constraint":  func() *korrel8.Constraint { return nil },
	"fail":        func(msg string) (string, error) { return "", errors.New(msg) },
	"toJSON":      ToJSON,
	"toYAML":      ToYAML,
	"k8sSelector": K8SSelector,
}

func Fail(message string) error {
	if message != "" {
		return fmt.Errorf("rule failed: %s", message)
	}
	return errors.New("rule failed")
}

func ToJSON(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
func ToYAML(v any) (string, error) { b, err := yaml.Marshal(v); return string(b), err }

func K8SSelector(m map[string]string) string {
	b := &strings.Builder{}
	sep := ""
	for k, v := range m {
		fmt.Fprintf(b, "%v%v=%v", sep, k, v)
		sep = ","
	}
	return b.String()
}
