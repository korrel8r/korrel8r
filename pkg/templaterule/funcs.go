package templaterule

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"sigs.k8s.io/yaml"
)

// FIXME document extra funcs, see text/template

// Funcs that are available in all templates created by New.
// Rule.Apply() adds the "constraint" function with the constraint if present.
var Funcs = map[string]any{
	"constraint":  func() *korrel8.Constraint { return nil },
	"has":         func(_ ...any) bool { return true }, // Used for side-effect: evaluate arguments to detect errors
	"assert":      doAssert,                            // Assert a condition in a template
	"toJSON":      toJSON,
	"toYAML":      toYAML,
	"fullname":    korrel8.FullName,
	"urlquerymap": urlQueryMap,
	"kvmap":       kvMap, // Useful for passing parameters to a template
}

var errFailed = errors.New("failed")

// assert a condition in a template - int return is a dummy value required for template functions.
func doAssert(ok bool, msg ...any) (int, error) {
	if !ok {
		if len(msg) > 0 {
			if s, ok := msg[0].(string); ok {
				return 0, fmt.Errorf(s, msg[1:]...)
			}
		}
		return 0, errFailed
	}
	return 0, nil
}

func toJSON(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
func toYAML(v any) (string, error) { b, err := yaml.Marshal(v); return string(b), err }

// urlQueryMap takes a map of any type and performs URL query encoding.
// Map values are stringified with fmt "%v"
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

func kvMap(keyValue ...any) map[any]any {
	m := map[any]any{}
	for i := 0; i < len(keyValue); i += 2 { // Will panic out of range on odd number
		m[keyValue[i]] = keyValue[i+1]
	}
	return m
}
