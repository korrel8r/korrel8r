// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var engineMeter = otel.Meter("korrel8r/engine")

var (
	metricStoreQueries, _       = engineMeter.Int64Counter("engine.store.queries", metric.WithDescription("Total store queries"))
	metricStoreQueryDuration, _ = engineMeter.Float64Histogram("engine.store.query.duration",
		metric.WithDescription("Store query duration in seconds"),
		metric.WithUnit("s"))
)
