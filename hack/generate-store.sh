#!/bin/bash
# Generate a directory-style mock store (file per query) from live cluster data.
# stdin: a korrel8r graph
# dir: directory containing files named for queries containing result from cluster.
#
# Example:
#   korrel8r objects "k8s:Pod:{}" | generate-store.sh testdata/mystore
#
test -e
DIR=${1:-"generated-store"}
mkdir -p "$DIR"
yq .nodes[].queries[].query | while read -r QUERY; do
	echo "Start: $QUERY"
	{
		korrel8r objects -o ndjson "$QUERY" >"$DIR/$QUERY"
		echo "Done: $QUERY"
	} &
done
wait
