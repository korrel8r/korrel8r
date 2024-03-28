# Variables common to Makefile and operator/Makefile

## VERSION: Semantic version for release. Use a -dev suffix for work in progress.
VERSION?=0.6.0

## OPERATOR_IMG: Base name of operator image.
OPERATOR_IMG?=quay.io/korrel8r/operator
OPERATOR_IMAGE?=$(OPERATOR_IMG):$(VERSION)

## KORREL8R_IMG: Base name of korrel8r image.
KORREL8R_IMG?=quay.io/korrel8r/korrel8r
KORREL8R_IMAGE?=$(KORREL8R_IMG):$(VERSION)

## NAMESPACE: Default namespace to deploy korrel8r.
NAMESPACE?=korrel8r

## IMGTOOL: May be podman or docker.
IMGTOOL?=$(shell which podman || which docker)

## ENVTEST_K8S_VERSION: version of kubebuilder for envtest testing.
ENVTEST_K8S_VERSION=1.29.x

## IMGTOOL: May be podman or docker.
IMGTOOL?=$(shell which podman || which docker)
