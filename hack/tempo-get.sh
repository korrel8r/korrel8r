#!/bin/bash
QUERY=${1:-'{ resource.service.name="article-service" }'}
URL=${2:-https://tempo-platform-gateway-openshift-tracing.apps.snoflake.home/api/traces/v1/platform/tempo/api/search}
curl -k -G --oauth2-bearer "$(oc whoami -t)" --data-urlencode "$QUERY" "$URL" | jq
