#!/bin/bash

set -e

GRAPH_JSON=$1
STORE_DIR=$2
if [ -z "$GRAPH_JSON" ] || [ -z "$STORE_DIR" ]; then
	  echo "Usage: $0 GRAPH_JSON STORE_DIR"
	  exit 1
fi

mkdir -p "$STORE_DIR"
yq .nodes[].queries[].query <"$GRAPH_JSON" | while read -r Q; do
    if [ -f "$STORE_DIR/$Q" ]; then
        echo "REPLACING: $Q"
    else
	      echo "$Q"
    fi
	  korrel8r objects "$Q" -o json >"$STORE_DIR/$Q"
done
