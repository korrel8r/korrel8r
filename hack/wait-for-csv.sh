#!/bin/bash
NAMESPACE=$1
NAME=$2				# CSV name without version

echo "waiting for $NAME to be created"
until CSV=$(oc get -n openshift-logging csv -o name | grep $NAME); do sleep 3; done
echo "waiting for $CSV to succeed"
oc wait --for=jsonpath='{.status.phase}'=Succeeded -n openshift-logging $CSV
