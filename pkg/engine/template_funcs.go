// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"errors"
	"fmt"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TemplateFuncs that are always available
//
// The standard [Go template] and [Sprig] library functions are always available, as are the following:
//
//	currentRule
//	  Returns the [korrel8r.Rule] currently being evaluated
//
//	require value
//	    Returns value if it is not nil, false, zero or empty, otherwise fails.
//	        Name is: {{ .metadata.name | require }}.
//
//	requireAll values...
//	    Fails if any value is nil, false, zero or empty.
//	        {{requireAll .metadata.namespace .metadata.name}}
//
//	query queryString
//	    Executes its argument as a korrel8r query string, returns the results as []any.
//	    May return an error.
//
//	k8sRouteHost namespace name
//	    Returns the spec.host of the named OpenShift route.
//	    Returns an error if the route is not found.
//
// [Sprig]: http://masterminds.github.io/sprig/
// [Go template]: https://pkg.go.dev/text/template#hdr-Functions
func (e *Engine) TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"query":        e.query,
		"k8sRouteHost": e.k8sRouteHost,
		"require":    func(v any) any { return rules.Require(v) },
		"requireAll": func(values ...any) string { rules.RequireAll(values...); return "" },
	}
}

// query implements the template function 'query'.
func (e *Engine) query(query string) ([]korrel8r.Object, error) {
	q, err := e.Query(query)
	if err != nil {
		return nil, err
	}
	results := result.New(q.Class())
	err = e.Get(context.Background(), q, nil, results)
	return results.List(), err
}

func (e *Engine) k8sRouteHost(namespace, name string) (host string, err error) {
	defer func() {
		if host == "" && err == nil {
			err = errors.New("not found")
		}
		if err != nil {
			err = fmt.Errorf("route/%v namespace=%v: %w", name, namespace, err)
		}
	}()
	query := fmt.Sprintf("k8s:Route.v1.route.openshift.io:{namespace: %v, name: %v}", namespace, name)
	routes, err := e.query(query)
	if err != nil || len(routes) == 0 {
		return "", err
	}
	host, _, err = unstructured.NestedString(routes[0].(k8s.Object), "spec", "host")
	return host, err
}
