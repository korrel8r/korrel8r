// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var mcpMeter = otel.Meter("korrel8r/mcp")

var (
	metricToolCalls, _    = mcpMeter.Int64Counter("mcp.tool.calls", metric.WithDescription("Total MCP tool calls"))
	metricToolDuration, _ = mcpMeter.Float64Histogram("mcp.tool.duration",
		metric.WithDescription("MCP tool call duration in seconds"),
		metric.WithUnit("s"))
)

// metrics returns middleware that records MCP tool call metrics.
func (s *Server) metrics(handler mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		start := time.Now()
		result, err := handler(ctx, method, req)
		duration := time.Since(start).Seconds()
		status := "ok"
		if err != nil {
			status = "error"
		} else if r, ok := result.(*mcp.CallToolResult); ok && r.IsError {
			status = "error"
		}
		attrs := metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("status", status))
		metricToolCalls.Add(ctx, 1, attrs)
		metricToolDuration.Record(ctx, duration, attrs)
		return result, err
	}
}
