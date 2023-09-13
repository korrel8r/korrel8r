#!/bin/bash
# Write a kustomization.yaml that will replace image $NAME with $NEW_NAME:$NEW_TAG
NAME=$1
NEW_NAME=$2
NEW_TAG=$3

cat <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: "${NAME}"
    newName: "${NEW_NAME}"
    newTag:  "${NEW_TAG}"
resources:
  - ../../base
EOF
