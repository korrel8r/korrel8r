// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

const sep = korrel8r.NameSeparator

func ClassString(c korrel8r.Class) string { return c.Domain().Name() + sep + c.Name() }

func ClassSplit(fullname string) (domain, class string) {
	s := strings.SplitN(fullname, sep, 2)
	if len(s) > 0 {
		domain = s[0]
	}
	if len(s) > 1 {
		class = s[1]
	}
	return domain, class
}

func QueryString(q korrel8r.Query) string { return ClassString(q.Class()) + sep + q.Data() }

func QuerySplit(query string) (domain, class, data string) {
	query = strings.TrimSpace(query)
	s := strings.SplitN(query, sep, 3)
	if len(s) > 0 {
		domain = s[0]
	}
	if len(s) > 1 {
		class = s[1]
	}
	if len(s) > 2 {
		data = s[2]
	}
	return domain, class, data
}
