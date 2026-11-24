// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
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
	// MetricExpr represents a vector selector; AppendString serializes it for match[].
	// This may canonicalize equivalent forms (e.g. __name__="up" becomes up).
	metricsql.VisitAll(e, func(node metricsql.Expr) {
		if vs, ok := node.(*metricsql.MetricExpr); ok {
			selectors = append(selectors, promSelector(vs))
		}
	})
	return selectors, nil
}

// errSuffix marks errors caused by malformed URL encoding or metric-specific
// parameter validation, as distinct from PromQL parse errors. Domain query parsing
// rejects only suffix errors: PromQL validity is deferred to Selectors.
var errSuffix = errors.New("invalid metric query suffix")

const namespaceParam = "namespace"

var namespaceRE = regexp.MustCompile(`\A[a-zA-Z0-9.-]+\z`) // decoded non-empty namespace value
var promMetricNameRE = regexp.MustCompile(`\A[a-zA-Z_:][a-zA-Z0-9_:]*\z`)

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

// promSelector avoids MetricsQL's escaped identifier syntax in Prometheus match[].
// Prometheus accepts arbitrary metric names as __name__ label values, but not as
// bare metric identifiers (e.g. MetricsQL escapes the hyphen in foo-bar).
func promSelector(vs *metricsql.MetricExpr) string {
	if len(vs.LabelFilterss) == 1 && len(vs.LabelFilterss[0]) > 0 {
		filters := vs.LabelFilterss[0]
		name := filters[0]
		if name.Label == "__name__" && !name.IsNegative && !name.IsRegexp &&
			name.Value != "" && !promMetricNameRE.MatchString(name.Value) {
			b := []byte{'{'}
			for i := range filters {
				if i > 0 {
					b = append(b, ',')
				}
				b = filters[i].AppendString(b)
			}
			return string(append(b, '}'))
		}
	}
	return string(vs.AppendString(nil))
}

func parseExpr(s string) (metricsql.Expr, error) {
	// Use MetricsQL instead of the Prometheus server parser to avoid pulling its
	// large dependency graph into this module. We only extract selectors, not
	// evaluate the expression. MetricsQL also accepts OR-separated label filter
	// groups, which Prometheus cannot use as match[].
	e, err := metricsql.Parse(s)
	if err != nil {
		return nil, err
	}
	var unsupported error
	metricsql.VisitAll(e, func(node metricsql.Expr) {
		if vs, ok := node.(*metricsql.MetricExpr); ok && len(vs.LabelFilterss) > 1 {
			unsupported = fmt.Errorf("PromQL does not support OR-separated label filters in %s", vs.AppendString(nil))
		}
	})
	if unsupported != nil {
		return nil, unsupported
	}
	return e, nil
}
