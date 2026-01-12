#!/bin/bash
# Extract queries from JSON result graph(s)
jq -r '.nodes[].queries[].query'
