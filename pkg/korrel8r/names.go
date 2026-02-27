// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"fmt"
	"regexp"
)

var (
	// labelRE for domain and class names. Disallow ':', space and URL-unsafe characters
	labelRE = regexp.MustCompile(`[^:\s<>#%{}|\^\[\]]+`)
	classRE = regexp.MustCompile(fmt.Sprintf("^(%v):(%v$)", labelRE, labelRE))
	queryRE = regexp.MustCompile(fmt.Sprintf("^(%v):(%v):(.*)$", labelRE, labelRE))
)

func ClassSplit(fullname string) (domain, class string, err error) {
	m := classRE.FindStringSubmatch(fullname)
	if len(m) == 0 {
		return "", "", fmt.Errorf("invalid class name: %v", fullname)
	}
	return m[1], m[2], nil
}

func QuerySplit(query string) (domain, class, data string, err error) {
	m := queryRE.FindStringSubmatch(query)
	if len(m) == 0 {
		return "", "", "", fmt.Errorf("invalid query: %v", query)
	}
	return m[1], m[2], m[3], nil
}

func ClassJoin(domain, name string) string {
	return fmt.Sprintf("%v:%v", domain, name)
}

func QueryJoin(domain, class, selector string) string {
	return fmt.Sprintf("%v:%v:%v", domain, class, selector)
}

func ClassString(c Class) string {
	return fmt.Sprintf("%v:%v", c.Domain(), c.Name())
}

func QueryString(q Query) string {
	return fmt.Sprintf("%v:%v", ClassString(q.Class()), q.Data())
}
