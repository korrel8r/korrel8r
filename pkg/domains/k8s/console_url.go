package k8s

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var consolePathRe = regexp.MustCompile(`(?:/k8s|/search)(?:(?:/ns/([^/]+))|/cluster|/all-namespaces)(?:/([^/]+)(?:/([^/]+))?)?`)

func parsePath(u *url.URL) (namespace, resource, name string, err error) {
	m := consolePathRe.FindStringSubmatch(u.Path)
	if m == nil {
		return "", "", "", fmt.Errorf("invalid k8s console URL: %q", u)
	}
	return m[1], m[2], m[3], nil
}

// Parse a group~version~Kind string
func parseGVK(gvkStr string) schema.GroupVersionKind {
	if p := strings.Split(gvkStr, "~"); len(p) == 3 {
		return schema.GroupVersionKind{Group: p[0], Version: p[1], Kind: p[2]}
	} else {
		return schema.GroupVersionKind{Kind: gvkStr}
	}
}

func parseSelector(s string) (map[string]string, error) {
	m := map[string]string{}
	s, err := url.QueryUnescape(s)
	if err != nil {
		return nil, err
	}
	for _, kv := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid selector string: %q", s)
		}
		m[k] = v
	}
	return m, nil
}

func selectorString(m map[string]string) string {
	var kvs []string
	for k, v := range m {
		kvs = append(kvs, fmt.Sprintf("%v=%v", k, v))
	}
	slices.Sort(kvs) // Predictable order
	return strings.Join(kvs, ",")
}
