// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"strconv"
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

// Metrics returns gin middleware that records HTTP request metrics.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		metricActiveRequests.Add(ctx, 1)
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		attrs := metric.WithAttributes(
			attribute.String("method", c.Request.Method),
			attribute.String("path", c.FullPath()),
			attribute.String("status", strconv.Itoa(c.Writer.Status())))
		metricRequests.Add(ctx, 1, attrs)
		metricRequestDuration.Record(ctx, duration, attrs)
		metricActiveRequests.Add(ctx, -1)
	}
}
