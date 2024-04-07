#!/bin/bash

set -e -o pipefail

NAMESPACE=$1                    # Subscription namespace
NAME=$2                         # Subscription name

until CSV=$(oc get -n $NAMESPACE subscription/$NAME -o jsonpath='{.status.currentCSV}') && [ -n "$CSV" ]; do
  echo "waiting for CSV for subscription/$NAME -n $NAMESPACE"
  sleep 1
done
until [ -n $(oc get -n $NAMESPACE csv/$CSV -o jsonpath='{.status.phase}') ] ; do
  echo "waiting for csv/$CSV status -n $NAMESPACE"
  sleep 1
done

echo "waiting for $CSV to have phase Succeeded"
oc wait --allow-missing-template-keys=true --for=jsonpath='{.status.phase}'=Succeeded -n $NAMESPACE csv/$CSV
