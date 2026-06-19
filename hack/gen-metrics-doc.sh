#!/bin/bash
# Generate markdown documentation for OTel metrics defined in metrics.go files.
# Output is a markdown document (without Hugo front matter).

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

cat <<'HEADER'
Korrel8r exposes [Prometheus](https://prometheus.io/) metrics at `/metrics`.
Scrape this endpoint or use the `--otel-collector` flag to push metrics via OTLP.

HEADER

# Join lines ending with a comma onto the next line, so multi-line metric
# declarations become single logical lines.
join_continuations() {
	sed -e ':a' -e '/,$/N; s/,\n/,/; ta' "$1"
}

# Find all metrics.go files under pkg/
find pkg -name metrics.go -print0 | sort -z | while IFS= read -r -d '' file; do
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
