// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// ClassName returns the fully qualified 'DOMAIN:CLASS' name of a class, e.g. "k8s:Pod.v1"
func ClassName(c Class) string {
	if c == nil {
		return "<nil>"
	}
	return c.Domain().Name() + ":" + c.Name()
}

// SplitClassName splits a fully qualified 'domain:class' name into class and domain.
func SplitClassName(fullname string) (class, domain string) {
	d, c, _ := strings.Cut(fullname, ":")
	return d, c
}

// RuleName returns a string including the rule name with full start and goal class names.
func RuleName(r Rule) string {
	return fmt.Sprintf("%v [%v]->[%v]", r.Name(), ClassName(r.Start()), ClassName(r.Goal()))
}

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
