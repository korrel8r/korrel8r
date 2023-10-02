// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// ClassName returns the fully qualified 'class.domain' name of a class, e.g. "Pod.v1.k8s"
func ClassName(c Class) string {
	if c == nil {
		return "<nil>"
	}
	name, domain := c.Name(), c.Domain().Name()
	if strings.HasSuffix(name, ".") {
		return name + domain
	}
	return fmt.Sprintf("%v.%v", name, domain)
}

// SplitClassName splits a fully qualified 'class.domain' name into class and domain.
// If there is no '.' treat the string as a domain name.
func SplitClassName(fullname string) (class, domain string) {
	i := strings.LastIndex(fullname, ".")
	if i < 0 {
		return "", fullname
	} else {
		return fullname[:i], fullname[i+1:]
	}
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
