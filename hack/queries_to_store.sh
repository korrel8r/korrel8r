#!/bin/bash
# Convert queries to a mock store YAML file.
# Use to create mock store files for off-line testing, based on real cluster data.
# Will sort and uniq the input.
sort -u | while read -r QUERY; do
	# Key must be quoted.
	echo "$QUERY" | jq -Rr '@json + ":"'
	korrel8r objects "$QUERY"
done
