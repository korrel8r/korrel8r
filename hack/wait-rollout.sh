#!/bin/bash

n=0
until oc get "$@" > /dev/null; do
  echo "waiting for resources: $*"
  sleep 5
  [ $(( ++n )) = 6 ] && { echo "timed out waiting for $*"; return 1; }
done
echo "waiting for rollout status: $*"
oc rollout status --watch --timeout=5m "$@" || exit 1
