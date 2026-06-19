// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("korrel8r/traverse")

var (
	metricRules, _            = meter.Int64Counter("traverse.rules", metric.WithDescription("Number of rule applications"))
	metricQueries, _          = meter.Int64Counter("traverse.queries", metric.WithDescription("Number of query executions"))
	metricDuplicateQueries, _ = meter.Int64Counter("traverse.duplicate_queries", metric.WithDescription("Number of duplicate queries ignored"))
)
