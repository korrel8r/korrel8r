#!/bin/bash
# Read  an PNG format icon from stdin, write a kustomize .yaml patch that sets the CSV icon field to stdout.
# Recommended icon size is 80x40
if [ -f $IN ] ; then
    cat <<EOF
# This patch adds an icon image to the CSV.
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: korrel8r
spec:
  icon:
    - mediatype: image/png
      base64data: >
EOF
    echo -n "        "
    base64 -w0
    echo
fi
