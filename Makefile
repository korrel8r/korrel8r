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

tools:				## Install tools for `make generate`.
	go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

VERSION_TXT=cmd/korrel8r/version.txt

generate:  $(VERSION_TXT) pkg/api/docs ## Run code generation, pre-build.
	hack/copyright.sh
	go mod tidy

$(VERSION_TXT): force
	@if test "$$(cat $(VERSION_TXT))" != "$(TAG)"; then echo $(TAG) | tee $@; fi

pkg/api/docs: $(shell find pkg/api pkg/korrel8r -name *.go)
	swag init -q -g $(dir $@)/api.go -o $@
	swag fmt $(dir $@)
	swagger -q generate markdown -f $@/swagger.json doc --output doc/rest-api.md

lint:				## Run the linter to find possible errors.
	golangci-lint run --fix

build:				## Build the korrel8r binary.
	go build -tags netgo ./cmd/korrel8r

test:				## Run all tests, requires a cluster.
	TEST_NO_SKIP=1 go test -timeout=1m -race ./...

cover:				## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

run: $(VERSION_TXT)             ## Run from source using checked-in default configuration.
	go run ./cmd/korrel8r/ web -c etc/korrel8r/korrel8r.yaml

## Build and deploy an image

IMAGE=$(IMG):$(TAG)

image: ## Build and push image. Set IMG to a writable, _public_ repository.
	$(IMGTOOL) build --tag=$(IMAGE) .
	$(IMGTOOL) push -q $(IMAGE)
	@echo $(IMAGE)

image-name:			## Print the full image name and tag.
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

release:	      ## Create a release tag, update changelog, push commit, push images.
	@echo "$(TAG)" | grep -qE "^v[0-9]+\.[0-9]+\.[0-9]+$$" || { echo "TAG=$(TAG) must be like vX.Y.Z"; exit 1; }
	@test -z "$$(git diff main origin/main)" || { echo "local main does not match origin"; exit 1; }
	make $(VERSION_TXT)	# Update version
	hack/changelog.sh $(VERSION_TXT) > CHANGELOG.md	# Update change log
	git commit -a -m "Release $(TAG)"     # Commit new release
	git tag $(TAG) -a -m "Release $(TAG)" # Tag the release
	git push origin main $(TAG)
	$(MAKE) image-latest

image-latest: image 		# Build and push the image and a "latest" alias
	$(IMGTOOL) push "$(IMAGE)" "$(IMG):latest"

.PHONY: force # Dummy target that is never satisfied
