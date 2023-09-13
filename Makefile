# Image name, without any version tag.
IMG?=quay.io/alanconway/korrel8r
# Image version tag, a semantic version of the form: vX.Y.Z-extras
TAG?=$(shell git describe)
# Kustomize overlay to use for `make deploy`
OVERLAY?=replace-image

# Use podman or docker, whichever is available.
IMGTOOL?=$(shell which podman || which docker)


help:				## Help for make targets
	@echo
	@echo = Make targets =
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

tools:	     			## Install tools used to generate code and documentation.
	go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

all: generate lint test image	## Generate code, lint, run tests, build image.

generate:			## Run code generation, pre-build.
	go generate -x ./...
	echo -e 'package main\nfunc Version() string { return "$(TAG)"; }' > cmd/korrel8r/version.go
	hack/copyright.sh

lint:				## Run the linter to find possible errors.
	golangci-lint run --fix

build:				## Build the korrel8r binary.
	go build -tags netgo ./cmd/korrel8r

.PHONY: test
test:				## Run all tests, requires a cluster.
	TEST_NO_SKIP=1 go test -timeout=1m -race ./...

cover:				## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

IMAGE=$(IMG):$(TAG)

image:				## Build and push a korrel8r image. Set IMG=your-image TAG=your-tag to change image.
	$(IMGTOOL) build --tag=$(IMAGE) .
	$(IMGTOOL) push -q $(IMAGE)
	@echo $(IMAGE)

image-name:			## Print the image name.
	@echo $(IMAGE)

IMAGE_KUSTOMIZATION=config/overlays/replace-image/kustomization.yaml
$(IMAGE_KUSTOMIZATION): force
	mkdir -p $(dir $@)
	hack/replace-image.sh REPLACE_ME $(IMG) $(TAG) > $@

WATCH=kubectl get events -A --watch-only& trap "kill %%" EXIT;

deploy:	$(IMAGE_KUSTOMIZATION) ## Create a new korrel8r deployment. Set 'IMG=your-image TAG=your-tag' to change image.
	$(WATCH) kubectl apply -k config/overlays/$(OVERLAY)
	$(WATCH) kubectl wait -n korrel8r --for=condition=available deployment.apps/korrel8r
	which oc >/dev/null && oc delete --ignore-not-found route/korrel8r && oc expose -n korrel8r svc/korrel8r

route-url:
	@oc get -n korrel8r route/korrel8r -o template='http://{{.spec.host}}'; echo

ifeq ($(TAG),$(shell git describe))
SAMETAG=1
endif
release:	## Create a release tag on the current branch. Set TAG=vX.Y.Z
	true $(if $(SAMETAG),$(error "TAG=$(TAG) is the current tag. TAG should be set to the release version"),)
	go mod tidy
	$(MAKE) all
	if [[ `git status --porcelain` ]]; then git status; $(error "git repository is not clean for relase"); fi
	git tag $(TAG) -a -m "Release $(TAG)"

deploy-latest: 	## Deploy the latest tagged release.
	$(MAKE) TAG=$(shell git tag) deploy

force:
