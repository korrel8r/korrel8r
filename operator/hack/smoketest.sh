#!/bin/bash
# Smoke test for a deployed Korrel8r operator.
# - Create a namespace and default Korrel8r instance
# - Wait for Korrel8r to be ready
# - Connect to the REST API, check the response looks OK.
#

NAMESPACE=${1:-korrel8r}
NAME=${2:-korrel8r}

set -e

oc apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: $NAMESPACE
---
apiVersion: korrel8r.openshift.io/v1alpha1
kind: Korrel8r
metadata:
  name: $NAME
  namespace: $NAMESPACE
spec:
  verbose: 9
EOF

oc wait -n $NAMESPACE korrel8r/$NAME --for=condition=Available --timeout 60s

HOST=$(oc get -n $NAMESPACE route/$NAME -o template='{{.spec.host}}')
URL=https://$HOST/api/v1alpha1/domains
RESPONSE=$(curl -k -f -sS --retry 10 --retry-all-errors --retry-connrefused $URL)

if [[ $RESPONSE =~ '"name":"k8s"' ]] ; then
    echo Success.
else
    cat <<EOF
Bad response from $URL:
$RESPONSE
EOF
    exit 1
fi
