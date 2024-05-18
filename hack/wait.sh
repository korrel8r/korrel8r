#!/usr/bin/env bash

set -e -o pipefail

declare RETRY_LIMIT=${RETRY_LIMIT:-6}
declare RETRY_DELAY=${RETRY_DELAY:-5}
declare -r ROLLOUT_TIMEOUT=5m

# Wait for a subscription to have a CSV with phase=succeeded.
subscription() {
	local ns=$1
	shift 1
	local csv=""
	for NAME in "$@"; do
		wait_for_resource "$ns" get subscription/"$NAME" -o jsonpath='{.status.currentCSV}' || return 1
		csv=$(kubectl get -n "$ns" subscription/"$NAME" -o jsonpath='{.status.currentCSV}')
		wait_for_resource "$ns" get csv/"$csv" -o jsonpath='{.status.phase}' || return 1
		oc wait --allow-missing-template-keys=true --for=jsonpath='{.status.phase}'=Succeeded -n "$ns" csv/"$csv" || return 1
	done
}

# Wait for a specific condition in a resource.
wait_for_resource() {
	local ns=$1
	local cmd=$2
	shift 2
	echo "Waiting for $* to be satisfied in $ns"
	local -i tries=0
	while ! kubectl "$cmd" -n "$ns" "$@" >/dev/null && [[ $tries -lt $RETRY_LIMIT ]]; do
		tries=$((tries + 1))
		echo "...[$tries / $RETRY_LIMIT]: waiting for $* to be satisfied in $ns"
		sleep "$RETRY_DELAY"
	done
	kubectl "$cmd" -n "$ns" "$@" >/dev/null || {
		echo "failed to get $* in $ns"
		return 1
	}
	return 0
}

# Wait for a workload to roll out.
rollout() {
	local ns=$1
	shift 1
	wait_for_resource "$ns" get "$@" || return 1
	echo "waiting for rollout status: $*"
	wait_for_resource "$ns" rollout status --watch --timeout="$ROLLOUT_TIMEOUT" "$@" || return 1
}

# Show usage.
show_usage() {
	echo "Usage: $0 {subscription|rollout} [NAMESPACE] [RESOURCE...]"
}

main() {
	[[ "$#" -lt 1 ]] && {
		show_usage
		return 1
	}
	local op=$1
	shift
	case "$op" in
	subscription | rollout)
		kubectl get events -A --watch-only &
		trap "kill %%" EXIT
		"$op" "$@"
		return $?
		;;
	*)
		show_usage
		return 1
		;;
	esac
}

main "$@"
