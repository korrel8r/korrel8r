#!/bin/bash
KORREL8R_CONFIG=$(git root)/etc/korrel8r/openshift-route.yaml
cat <<EOF
{
  "mcpServers": {
    "korrel8r": {
      "command": "korrel8r",
      "args": [
        "mcp",
        "--config", "$KORREL8R_CONFIG"
      ]
    }
  }
}
EOF
