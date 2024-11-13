#!/bin/bash
# Enable the troubleshooting panel in the openshift console.
# Temporary work-around till the observability operator does this for us.

REPO=${REPO:-github.com:openshift/troubleshooting-panel-console-plugin}
IMG=${IMG:-quay.io/alanconway/troubleshooting-panel-console-plugin}

git remote -v | grep -q "$REPO" || {
	echo "Must run in a clone of $REPO"
	exit 1
}

podman build -t "$IMG" .
podman push "$IMG"

helm upgrade -i troubleshooting-panel-console-plugin charts/openshift-console-plugin -n troubleshooting-panel-console-plugin --create-namespace --set plugin.image="$IMG"
