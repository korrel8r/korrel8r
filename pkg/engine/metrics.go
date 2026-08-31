// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var engineMeter = otel.Meter("korrel8r/engine")

var (
	metricStoreQueries, _       = engineMeter.Int64Counter("engine.store.get", metric.WithDescription("Total store get calls"))
	metricStoreQueryDuration, _ = engineMeter.Float64Histogram("engine.store.get.duration",
		metric.WithDescription("Store get duration in seconds"),
		metric.WithUnit("s"))
)
