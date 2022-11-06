package templaterule

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"sigs.k8s.io/yaml"
)

// FIXME document extra funcs, see text/template

// Funcs that are available in templates created by New.
// Rule.Apply() adds the "constraint" function with the constraint if present.
var Funcs = map[string]any{
	"constraint":  func() *korrel8.Constraint { return nil },
	"has":         func(_ ...any) bool { return true },
	"json":        toJSON,
	"yaml":        toYAML,
	"fullname":    korrel8.FullName,
	"urlquerymap": urlQueryMap,
}

func toJSON(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
func toYAML(v any) (string, error) { b, err := yaml.Marshal(v); return string(b), err }

func doAssert(ok bool) string {
	if !ok {
		panic("assertion failed")
	}
	return ""
}

func urlQueryMap(m any) string {
	v := reflect.ValueOf(m)
	if !v.IsValid() {
		return ""
	}
	p := url.Values{}
	i := v.MapRange()
	for i.Next() {
		p.Add(fmt.Sprintf("%v", i.Key()), fmt.Sprintf("%v", i.Value()))
	}
	return p.Encode()
}
