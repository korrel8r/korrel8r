# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Print this help message.
	@echo; echo = Targets =
	@grep -E '^\w+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z_]+: ' Makefile | sed 's/^## \([A-Z_]*\): \(.*\)/\1#\2/' | column -s'#' -t


# The following variables can be overridden by environment variables or on the `make` command line

## IMG: Name of image to build or deploy, without version tag.
IMG?=quay.io/korrel8r/korrel8r
## TAG: Version tag of image, a semantic version like vX.Y.Z-extras.
TAG?=$(shell git describe)
## OVERLAY: Name of kustomize directory in config/overlays to use for `make deploy`.
OVERLAY?=dev
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(shell which podman || which docker)

all: generate lint test install	## Local code validation: generate, lint, test, build and install.

tools: ## Install tools for `make generate` and `make lint` locally.
	go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

VERSION_TXT=cmd/korrel8r/version.txt
generate: $(VERSION_TXT) pkg/api/docs ## Run code generation, pre-build.
	hack/copyright.sh
	go mod tidy

ifneq ($(TAG),$(file <$(VERSION_TXT))) # VERSION_TXT does not match TAG
$(VERSION_TXT): force
	echo $(TAG) | tee $@
	sed 's/^:version:.*/:version: $(TAG)/' -i docs/index.adoc
endif

pkg/api/docs: $(shell find pkg/api pkg/korrel8r -name *.go)
	swag init -q -g $(dir $@)/api.go -o $@
	swag fmt $(dir $@)
	swagger -q generate markdown -f pkg/api/docs/swagger.json --output $@/swagger.md
	which pandoc > /dev/null && pandoc $@/swagger.md -o docs/rest-api.adoc

lint: ## Run the linter to find and fix code style problems.
	golangci-lint run --fix

install: ## Build and install the korrel8r binary locally in $GOBIN.
	go install -tags netgo ./cmd/korrel8r

test: ## Run all tests, requires a cluster.
	TEST_NO_SKIP=1 go test -timeout=1m -race ./...

cover: ## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

CONFIG=etc/korrel8r/korrel8r.yaml
run: $(VERSION_TXT) ## Run `korrel8r web` from source using configuration in ./etc.
	go run ./cmd/korrel8r/ web -c $(CONFIG)

IMAGE=$(IMG):$(TAG)
image: ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) build --tag=$(IMAGE) .
	$(IMGTOOL) push -q $(IMAGE)
	@echo $(IMAGE)

image-name: ## Print the full image name and tag.
	@echo $(IMAGE)

IMAGE_KUSTOMIZATION=config/overlays/$(OVERLAY)/kustomization.yaml
$(IMAGE_KUSTOMIZATION): force
	mkdir -p $(dir $@)
	hack/replace-image.sh "quay.io/korrel8r/korrel8r" $(IMG) $(TAG) > $@

WATCH=kubectl get events -A --watch-only& trap "kill %%" EXIT;

deploy: $(IMAGE_KUSTOMIZATION)	## Deploy to a cluster using kustomize. IMG must be set to a *public* image repository.
	$(WATCH) kubectl apply -k config/overlays/$(OVERLAY)
	$(WATCH) kubectl wait -n korrel8r --for=condition=available deployment.apps/korrel8r

OC_GET_ROUTE_URL=oc get -n korrel8r route/korrel8r -o template='http://{{.spec.host}}{{"\n"}}'
route: ## Print URL of route to korrel8r deployed in a cluster (requires openshift cluster)
	$(OC_GET_ROUTE_URL) || { oc expose -n korrel8r svc/korrel8r && $(OC_GET_ROUTE_URL); }

release: ## Create a release tag, update changelog, push commit, push images.
	@echo "$(TAG)" | grep -qE "^v[0-9]+\.[0-9]+\.[0-9]+$$" || { echo "TAG=$(TAG) must be like vX.Y.Z"; exit 1; }
	@test -z "$$(git diff main origin/main)" || { echo "local main does not match origin"; exit 1; }
	make $(VERSION_TXT)	# Update version
	hack/changelog.sh $(TAG) > CHANGELOG.md	# Update change log
	git commit -a -m "Release $(TAG)"     # Commit new release
	git tag $(TAG) -a -m "Release $(TAG)" # Tag the release
	git push origin $(TAG)
	git push origin main
	$(MAKE) image-latest	# Only build & push images after git tagging succeeds

image-latest: image
	$(IMGTOOL) push "$(IMAGE)" "$(IMG):latest"

.PHONY: force # Dummy target that is never satisfied
