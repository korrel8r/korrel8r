// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/prometheus/prometheus/promql/parser"
)

// Query is a [PromQL] query string.
//
// Korrel8r uses metric labels for correlation, it does not use time-series data values.
// The [PromQL] query is analyzed to identify series it uses,
// labels of those series are used for correlation.
//
// [PromQL]: https://prometheus.io/docs/prometheus/latest/querying/basics/
type Query string

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return korrel8r.QueryString(q) }

func (q Query) Selectors() ([]string, error) {
	var selectors []string
	expr, err := parser.ParseExpr(string(q))
	if err != nil {
		return nil, err
	}
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			selectors = append(selectors, vs.String())
		}
		return nil
	})
	return selectors, err
}
