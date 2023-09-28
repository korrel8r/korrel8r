# Image name without version tag.
IMG?=quay.io/korrel8r/korrel8r
# Image version tag, a semantic version of the form: vX.Y.Z-extras
TAG?=$(shell git describe)
# Kustomize overlay to use for `make deploy`.
OVERLAY?=replace-image

# Use podman or docker, whichever is available.
IMGTOOL?=$(shell which podman || which docker)

## Local build and test

help:				## Help for make targets
	@echo
	@echo Make targets; echo
	@grep ':.*\s##' Makefile | sed 's/:.*##/:/' | column -s: -t

all: generate lint test	 	## Verify code changes: generate, lint, and test.

tools:	     			## Install tools used to generate code and documentation.
	go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

generate: pkg/api/docs		## Run code generation, pre-build.
	go mod tidy
	hack/copyright.sh
	echo $(TAG) > cmd/korrel8r/version.txt

pkg/api/docs: $(shell find pkg/api pkg/korrel8r -name *.go)
	swag init -q -g $(dir $@)/api.go -o $@
	swag fmt $(dir $@)
	cp $@/swagger.json doc
	swagger -q generate markdown -f doc/swagger.json --output doc/rest-api.md

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

run: 				## Run from source using checked-in default configuration.
	go run ./cmd/korrel8r/ web -c etc/korrel8r/korrel8r.yaml

## Build and deploy an image

IMAGE=$(IMG):$(TAG)

image:				## Build and push a korrel8r image. Set IMG to you _public_ image repository, e.g. IMG=quay.io/myquayaccount/korrel8r
	$(IMGTOOL) build --tag=$(IMAGE) .
	$(IMGTOOL) push -q $(IMAGE)
	@echo $(IMAGE)

image-name:			## Print the image name with tag.
	@echo $(IMAGE)

IMAGE_KUSTOMIZATION=config/overlays/replace-image/kustomization.yaml
$(IMAGE_KUSTOMIZATION): force
	mkdir -p $(dir $@)
	hack/replace-image.sh REPLACE_ME $(IMG) $(TAG) > $@

WATCH=kubectl get events -A --watch-only& trap "kill %%" EXIT;

deploy: $(IMAGE_KUSTOMIZATION)	## Deploy to a cluster using customize.
	$(WATCH) kubectl apply -k config/overlays/$(OVERLAY)
	$(WATCH) kubectl wait -n korrel8r --for=condition=available deployment.apps/korrel8r
	which oc >/dev/null && oc delete --ignore-not-found route/korrel8r && oc expose -n korrel8r svc/korrel8r

route-url:			## URL of route to korrel8r on cluster (requires openshift for route)
	@oc get route/korrel8r -o template='http://{{.spec.host}}'; echo


## Create a release
VERSION_TXT=cmd/korrel8r/version.txt
check-tag:
	@echo "$(TAG)" | grep -qE "^v[0-9]+\.[0-9]+\.[0-9]+$$" || { echo "TAG=$(TAG) must be of the form vX.Y.Z"; exit 1; }
release: check-tag		## Create a release tag and commit, push images.
	$(MAKE) all TAG=$(TAG)
	@if git status --porcelain | grep -v "M $(VERSION_TXT)"; then				\
		echo "git repository is dirty, only $(VERSION_TXT) should be modified"; exit 1;	\
	fi
	git commit -a -m "Release $(TAG)"
	git tag $(TAG) -a -m "Release $(TAG)"
	git push origin $(TAG)
	$(MAKE) image
	$(IMGTOOL) push "$(IMAGE)" "$(IMG):latest"

force:
