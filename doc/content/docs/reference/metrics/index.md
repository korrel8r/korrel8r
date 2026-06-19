---
title: Metrics
description: Prometheus metrics reference
weight: 20
---
<!-- Generated content, do not edit! -->
Korrel8r exposes [Prometheus](https://prometheus.io/) metrics at `/metrics`.
Scrape this endpoint or use the `--otel-collector` flag to push metrics via OTLP.

## korrel8r/engine

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `engine.store.queries` | counter |  | Total store queries |
| `engine.store.query.duration` | histogram | s | Store query duration in seconds |

## korrel8r/traverse

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `traverse.rules` | counter |  | Number of rule applications |
| `traverse.queries` | counter |  | Number of query executions |
| `traverse.duplicate_queries` | counter |  | Number of duplicate queries ignored |

## korrel8r/mcp

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `mcp.tool.calls` | counter |  | Total MCP tool calls |
| `mcp.tool.duration` | histogram | s | MCP tool call duration in seconds |

## korrel8r/rest

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `rest.requests` | counter |  | Total HTTP requests |
| `rest.request.duration` | histogram | s | HTTP request duration in seconds |
| `rest.active.requests` | gauge |  | In-flight HTTP requests |

