// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"strings"
)

// NameSeparator used in DOMAIN:CLASS and DOMAIN:CLASS:QUERY strings.
const NameSeparator = ":"

func join(strs ...string) string { return strings.Join(strs, NameSeparator) }

func split(str string, n int) []string {
	return strings.SplitN(strings.TrimSpace(str), NameSeparator, n)
}

func ClassString(c Class) string { return join(c.Domain().Name(), c.Name()) }

func ClassSplit(fullname string) (domain, class string) {
	s := split(fullname, 2)
	if len(s) > 0 {
		domain = s[0]
	}
	if len(s) > 1 {
		class = s[1]
	}
	return domain, class
}

func QueryString(q Query) string { return join(ClassString(q.Class()), q.Data()) }

func QuerySplit(query string) (domain, class, data string) {
	s := split(query, 3)
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
