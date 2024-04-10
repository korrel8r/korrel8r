#!/bin/bash
# Wait for resources and conditions

set -e -o pipefail

RETRY_LIMIT=${RETRY_LIMIT:-6}
RETRY_DELAY=${RETRY_DELAY:-5s}
TIMEOUT=5m

retry() {
	until "$@"; do
		echo "waiting for: $*"
		sleep "$RETRY_DELAY"
		if [ $((++n)) = "$RETRY_LIMIT" ]; then
			echo "timed out: $*"
			return 1
		fi
	done
}

# Wait for resources to exist.
exists() {
	retry oc get "$@"
}

# Wait for a subscription to have a CSV with phase=succeeded.
subscription() {
	if [ "$1" = "-n" ]; then
		NS_FLAG="-n $2"
		shift
		shift
	fi
	for NAME in "$@"; do
		until CSV=$(oc get "$NS_FLAG" subscription/"$NAME" -o jsonpath='{.status.currentCSV}') && [ -n "$CSV" ]; do
			echo "waiting for CSV for subscription/$NAME $NS_FLAG"
			sleep "$RETRY_DELAY"
		done
		until [[ -n $(oc get -n "$NAMESPACE" csv/"$CSV" -o jsonpath='{.status.phase}') ]]; do
			echo "waiting for csv/$CSV status $NS_FLAG"
			sleep "$RETRY_DELAY"
		done
		echo "waiting for $CSV to have phase Succeeded"
		oc wait --allow-missing-template-keys=true --for=jsonpath='{.status.phase}'=Succeeded "$NS_FLAG" csv/"$CSV" || return 1
	done
}

# Wait for a workload to roll out.
rollout() {
	retry oc get "$@" >/dev/null || return 1
	echo "waiting for rollout status: $*"
	oc rollout status --watch --timeout="$TIMEOUT" "$@" || return 1
}

case "$1" in
exists | subscription | rollout)
	"$@"
	;;
*)
	echo "$0 [$CMDS] [-n NAMESPACE] [RESOURCE...]"
	exit 1
	;;
esac
