#!/bin/bash
# Generate markdown documentation for OTel metrics defined in metrics.go files.
# Output is a markdown document (without Hugo front matter).

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

cat <<'HEADER'
Korrel8r automatically exposes a metrics scrape endpoint for [Prometheus](https://prometheus.io/) at `/metrics`. Alternatively, run with the `--otel-collector` flag to push metrics via OTLP.

HEADER

# Join lines ending with a comma onto the next line, so multi-line metric
# declarations become single logical lines.
join_continuations() {
	sed -e ':a' -e '/,$/N; s/,\n/,/; ta' "$1"
}

# Find all metrics.go files under pkg/
find pkg internal/pkg -name metrics.go -print0 | sort -z | while IFS= read -r -d '' file; do
	meter=""

	while IFS= read -r line; do
		# Extract meter name
		if [[ "$line" =~ otel\.Meter\(\"([^\"]+)\"\) ]]; then
			meter="${BASH_REMATCH[1]}"
			echo "## ${meter}"
			echo
			echo "| Metric | Type | Unit | Description |"
			echo "|--------|------|------|-------------|"
			continue
		fi

		# Extract metric definitions
		if [[ "$line" =~ (Int64Counter|Int64UpDownCounter|Float64Histogram|Float64Counter|Int64Histogram)\(\"([^\"]+)\" ]]; then
			go_type="${BASH_REMATCH[1]}"
			name="${BASH_REMATCH[2]}"

			case "$go_type" in
			Int64Counter | Float64Counter) prom_type="counter" ;;
			Int64UpDownCounter) prom_type="gauge" ;;
			Int64Histogram | Float64Histogram) prom_type="histogram" ;;
			*) prom_type="$go_type" ;;
			esac

			unit=""
			if [[ "$line" =~ metric\.WithUnit\(\"([^\"]+)\"\) ]]; then
				unit="${BASH_REMATCH[1]}"
			fi

			desc=""
			if [[ "$line" =~ metric\.WithDescription\(\"([^\"]+)\"\) ]]; then
				desc="${BASH_REMATCH[1]}"
			fi

			echo "| \`${name}\` | ${prom_type} | ${unit} | ${desc} |"
		fi
	done < <(join_continuations "$file")

	if [[ -n "$meter" ]]; then
		echo
	fi
done

cat <<'EXAMPLES'
## Prometheus metric names

OTel metric names are converted for Prometheus export:
dots become underscores, counters get a `_total` suffix,
and histogram units are appended (e.g. `_seconds` for unit `s`).

For example `engine.store.get` becomes `engine_store_get_total`,
and `engine.store.get.duration` becomes `engine_store_get_duration_seconds`.

## Example PromQL queries

Store get rate per second by domain:

```promql
rate(engine_store_get_total[5m])
```

Store get error ratio by domain:

```promql
  rate(engine_store_get_total{status="error"}[5m])
/ rate(engine_store_get_total[5m])
```

Average store get latency by domain:

```promql
  rate(engine_store_get_duration_seconds_sum[5m])
/ rate(engine_store_get_duration_seconds_count[5m])
```

P99 store get latency:

```promql
histogram_quantile(0.99, rate(engine_store_get_duration_seconds_bucket[5m]))
```

REST request rate by method and status:

```promql
rate(rest_requests_total[5m])
```

Average REST request latency:

```promql
  rate(rest_request_duration_seconds_sum[5m])
/ rate(rest_request_duration_seconds_count[5m])
```

MCP tool call rate:

```promql
rate(mcp_tool_calls_total[5m])
```

Top 10 most-applied rules:

```promql
topk(10, increase(traverse_rules_total[1h]))
```

Graph traversal efficiency (duplicate queries as a fraction of total):

```promql
  rate(traverse_duplicate_queries_total[5m])
/ rate(traverse_queries_total[5m])
```
EXAMPLES
