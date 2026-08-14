#!/bin/bash
# pipe OTEL *-metric.json file into this expression to find highest-used rules.
jq    '[.ScopeMetrics[].Metrics[] | select(.Name == "traverse.rules") | .Data.DataPoints[] | {rule: (.Attributes[] | select(.Key == "rule").Value.Value), value: .Value}]  | unique_by(.value) '
