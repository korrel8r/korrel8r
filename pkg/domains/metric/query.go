// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/prometheus/promql/parser"
)

// Query is a [PromQL] expression with optional trailing namespace parameters.
//
//	PROMQL[?namespace=NS[&namespace=NS...]]
//
// Korrel8r parses the expression to identify its time series and uses their labels,
// not sample values, for correlation.
//
// For namespace-scoped queries, include both a selector for the metric's namespace
// label and a matching ?namespace=NS suffix. The label need not be named "namespace"
// (for example, it may be "exported_namespace"). The selector filters time series;
// the suffix specifies the scope and selects the configured namespaced URL. If no
// namespaced URL is configured, the store uses the regular metric URL instead.
// Queries without a suffix use the regular metric URL.
//
// [PromQL]: https://prometheus.io/docs/prometheus/latest/querying/basics/
type Query string

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return korrel8r.QueryString(q) }

// Params returns the parameters of the namespace suffix, or nil if q has no suffix.
// It returns an error for a malformed suffix.
func (q Query) Params() (url.Values, error) {
	_, params, err := splitQuery(string(q))
	return params, err
}

// Selectors returns the vector selectors of the [PromQL] expression part of q,
// excluding any namespace suffix.
func (q Query) Selectors() ([]string, error) {
	expr, _, err := splitQuery(string(q))
	if err != nil {
		return nil, err
	}
	e, err := parseExpr(expr)
	if err != nil {
		return nil, err
	}
	var selectors []string
	parser.Inspect(e, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			selectors = append(selectors, vs.String())
		}
		return nil
	})
	return selectors, nil
}

// errSuffix marks errors caused by malformed URL encoding or metric-specific
// parameter validation, as distinct from PromQL parse errors. Domain query parsing
// rejects only suffix errors: PromQL validity is deferred to Selectors.
var errSuffix = errors.New("invalid metric query suffix")

const namespaceParam = "namespace"

var namespaceRE = regexp.MustCompile(`\A[a-zA-Z0-9.-]+\z`) // decoded non-empty namespace value

// splitQuery first gives the complete string to PromQL, so question marks inside
// strings and comments remain part of the expression. If it is not valid PromQL,
// the shared helper looks for a valid URL query section from the back. Metric-specific
// parameter validation and expression parsing happen only after a section is found.
func splitQuery(s string) (expr string, params url.Values, err error) {
	if _, parseErr := parseExpr(s); parseErr == nil {
		return s, nil, nil
	} else {
		expr, params, err = impl.SplitURLQueryData(s)
		if err != nil {
			return "", nil, fmt.Errorf("%w: %w", errSuffix, err)
		}
		if params == nil {
			return "", nil, parseErr
		}
	}
	if err := validateParams(params); err != nil {
		return "", nil, err
	}
	if _, err := parseExpr(expr); err != nil {
		return "", nil, err
	}
	return expr, params, nil
}

// validateParams checks suffix parameters: namespace is the only allowed parameter
// and decoded values must be non-empty namespace names.
func validateParams(params url.Values) error {
	for k, vs := range params {
		if k != namespaceParam {
			return fmt.Errorf("%w: unknown parameter %q, only %q is allowed", errSuffix, k, namespaceParam)
		}
		for _, v := range vs {
			if v == "" {
				return fmt.Errorf("%w: empty %v value", errSuffix, k)
			}
			if !namespaceRE.MatchString(v) {
				return fmt.Errorf("%w: invalid %v value %q", errSuffix, k, v)
			}
		}
	}
	return nil
}

func parseExpr(s string) (parser.Expr, error) {
	return parser.NewParser(parser.Options{}).ParseExpr(s)
}
