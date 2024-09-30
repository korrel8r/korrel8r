#!/usr/bin/env bash

set -e -o pipefail

declare RETRY_LIMIT=${RETRY_LIMIT:-10}
declare RETRY_DELAY=${RETRY_DELAY:-10}
declare -r ROLLOUT_TIMEOUT=5m

# Wait for a subscription to have a CSV with phase=succeeded.
subscription() {
	local ns=$1
	shift 1
	local csv=""
	for NAME in "$@"; do
		wait_for_resource "subscription $NAME in $ns to be known" \
			check_condition "AtLatestKnown" kubectl -n "$ns" get subscription/"$NAME" -o jsonpath='{.status.state}' || return 1
		csv=$(kubectl get -n "$ns" subscription/"$NAME" -o jsonpath='{.status.currentCSV}')
		wait_for_resource "csv $csv to be available" check_condition "Succeeded" kubectl get -n "$ns" csv/"$csv" -o jsonpath='{.status.phase}' || return 1
		oc wait --allow-missing-template-keys=true --for=jsonpath='{.status.phase}'=Succeeded -n "$ns" csv/"$csv" || return 1
	done
}

check_condition() {
	local exp="$1"
	local cond="$2"
	shift 2
	[[ -n $exp ]] && [[ $("$cond" "$@") == "$exp" ]] && {
		return 0
	}
	return 1
}

# Wait for a specific condition in a resource.
wait_for_resource() {
	local msg="$1"
	local condition="$2"
	shift 2
	echo "Waiting for [$RETRY_LIMIT x $RETRY_DELAY s]: $msg"
	local -i tries=0
	local -i ret=1
	while [[ $tries -lt $RETRY_LIMIT ]]; do

		$condition "$@" && {
			ret=0
			break
		}
		tries=$((tries + 1))
		echo "...[$tries / $RETRY_LIMIT]: waiting for ($RETRY_DELAY s) - $msg" >&2
		sleep "$RETRY_DELAY"
	done

	return $ret
}

# Wait for a workload to roll out.
rollout() {
	local ns=$1
	shift 1
	wait_for_resource "deployments to be created" kubectl -n "$ns" get "$@" || return 1
	wait_for_resource "rollout to complete" kubectl -n "$ns" rollout status --watch --timeout="$ROLLOUT_TIMEOUT" "$@" || return 1
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
