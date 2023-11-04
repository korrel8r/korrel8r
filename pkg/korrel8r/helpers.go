// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// sep used in DOMAIN:CLASS and DOMAIN:CLASS:QUERY strings.
const sep = ":"

// ClassName returns the fully qualified 'DOMAIN:CLASS' name of a class, e.g. "k8s:Pod.v1"
func ClassName(c Class) string {
	if c == nil {
		return "<nil>"
	}
	return c.Domain().Name() + sep + c.Name()
}

// SplitClassName splits a DOMAIN:CLASS name into DOMAIN and CLASS.
func SplitClassName(fullname string) (domain, class string, ok bool) {
	return strings.Cut(fullname, sep)
}

// SplitClassData splits a DOMAIN:CLASS:DATA string into DOMAIN, CLASS and DATA.
// This form is used for queries and objects.
func SplitClassData(q string) (domain, class, data string, ok bool) {
	if domain, cq, ok := strings.Cut(q, sep); ok {
		class, data, ok := strings.Cut(cq, sep)
		return domain, class, data, ok
	}
	return "", "", "", false
}

// RuleName returns a string including the rule name with full start and goal class names.
func RuleName(r Rule) string {
	return fmt.Sprintf("%v(%v,%v)", r.Name(), ClassName(r.Start()), ClassName(r.Goal()))
}

// QueryName returns the full DOMAIN:CLASS:QUERY string form of a query.
func QueryName(q Query) string { return string(ClassName(q.Class()) + sep + q.Query()) }

// JSONString returns the JSON marshaled string from v, or the error message if marshal fails
func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", err.Error())
	}
	return string(b)
}

func YAMLString(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", err.Error())
	}
	return string(b)
}
