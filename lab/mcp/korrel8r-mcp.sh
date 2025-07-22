#!/bin/bash
# Run a korrel8r stdio MCP server.

cd $(dirname $0)
go run ../../cmd/korrel8r -v3 -c ../../etc/korrel8r/openshift-route.yaml mcp 2> $LOG
