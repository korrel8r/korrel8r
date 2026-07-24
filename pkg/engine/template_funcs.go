// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// # Template Functions
//
// These functions are available in all rule templates, in addition to the
// standard [Go template functions] and the [Sprig] library.
// Some domains provide additional domain-specific functions, documented in the domain reference.
//
//	assert [message] value
//	    Fails the template if value is empty.
//	    An optional message string is included in the error.
//	    A value is "empty" if it is nil, false, zero, or a zero-length string, slice, or map.
//	    Can also be used with piped syntax: {{ value | assert "message" }}.
//
//	required [message] value
//	    Like assert, but returns the value if it is not empty.
//	    Fails the template if the value is empty.
//	    Can also be used with piped syntax: {{ value | required "message" }}.
//
//	query queryString
//	    Executes its argument as a korrel8r query string, returns the results as []any.
//	    May return an error.
//
//	k8sRouteHost namespace name
//	    Returns the spec.host of the named OpenShift route.
//	    Returns an error if the route is not found.
//
// # Asserting Multiple Values
//
// Use the [Sprig] function "all" to assert that several values are non-empty.
// The "all" function returns true only when every argument is non-empty.
//
// Examples:
//
//	{{assert (all .metadata.namespace .metadata.name)}}
//	{{assert "need namespace, name and labels" (all .metadata.namespace .metadata.name .metadata.labels)}}
//
// Use "required" to pass through a value while also checking that related values are present:
//
//	{{required "need namespace" .metadata.namespace}}
//
// [Go template functions]: https://pkg.go.dev/text/template#hdr-Functions
// [Sprig]: http://masterminds.github.io/sprig/
package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// globalTemplateFuncs available at all times, including during configuration processing.
func globalTemplateFuncs(e *Engine) template.FuncMap {
	return template.FuncMap{
		"query":        e.query,
		"k8sRouteHost": e.k8sRouteHost,
		"assert":       templateAssert,
		"required":     templateRequired,
	}
}

// templateAssert fails template generation if value is not true.
// Usage: assert [message] value
func templateAssert(args ...any) (string, error) {
	switch len(args) {
	case 1:
		if !isEmpty(args[0]) {
			return "", nil
		}
		return "", fmt.Errorf("assertion failed")
	case 2:
		msg, ok := args[0].(string)
		if !ok {
			return "", fmt.Errorf("assert: message must be a string")
		}
		if !isEmpty(args[1]) {
			return "", nil
		}
		return "", fmt.Errorf("assertion failed: %s", msg)
	default:
		return "", fmt.Errorf("assert: expected 1 or 2 arguments, got %d", len(args))
	}
}

// templateRequired passes through value if non-empty and non-nil, otherwise fails.
// Usage: required [message] value
func templateRequired(args ...any) (any, error) {
	switch len(args) {
	case 1:
		if isEmpty(args[0]) {
			return nil, fmt.Errorf("a required value was not set")
		}
		return args[0], nil
	case 2:
		msg, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("required: message must be a string")
		}
		if isEmpty(args[1]) {
			return nil, fmt.Errorf("%s", msg)
		}
		return args[1], nil
	default:
		return nil, fmt.Errorf("required: expected 1 or 2 arguments, got %d", len(args))
	}
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	default:
		return rv.IsZero()
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
