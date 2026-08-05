// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var restMeter = otel.Meter("korrel8r/rest")

var (
	metricRequests, _        = restMeter.Int64Counter("rest.requests", metric.WithDescription("Total HTTP requests"))
	metricRequestDuration, _ = restMeter.Float64Histogram("rest.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"))
	metricActiveRequests, _ = restMeter.Int64UpDownCounter("rest.active.requests", metric.WithDescription("In-flight HTTP requests"))
)

type requestMetricKey struct {
	method, path, status string
}

type requestMetricAttrs struct {
	mu    sync.Mutex
	attrs map[requestMetricKey]metric.MeasurementOption
}

func (a *requestMetricAttrs) get(method, path, status string) metric.MeasurementOption {
	key := requestMetricKey{method: method, path: path, status: status}
	a.mu.Lock()
	defer a.mu.Unlock()
	if attrs, ok := a.attrs[key]; ok {
		return attrs
	}
	attrs := metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.String("status", status))
	a.attrs[key] = attrs
	return attrs
}

var cachedRequestAttrs = requestMetricAttrs{attrs: map[requestMetricKey]metric.MeasurementOption{}}

// Metrics returns gin middleware that records HTTP request metrics.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		metricActiveRequests.Add(ctx, 1)
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		attrs := cachedRequestAttrs.get(c.Request.Method, c.FullPath(), strconv.Itoa(c.Writer.Status()))
		metricRequests.Add(ctx, 1, attrs)
		metricRequestDuration.Record(ctx, duration, attrs)
		metricActiveRequests.Add(ctx, -1)
	}
}
