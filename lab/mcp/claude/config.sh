#!/bin/bash
CLAUDE_CONFIG=$HOME/.config/Claude/claude_desktop_config.json
$(dirname $0)/../mcpconfig-local.sh > $CLAUDE_CONFIG
echo "Wrote $CLAUDE_CONFIG"
cat $CLAUDE_CONFIG
