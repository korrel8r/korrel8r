# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release, use -dev for development pre-release versions.
VERSION?=0.7.0
## REGISTRY: Name of image registry
REGISTRY?=quay.io
## REGISTRY_ORG: Name of registry organization.
REGISTRY_ORG?=$(error Set REGISTRY_ORG to push or pull images)
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell podman info > /dev/null 2>&1 && which podman), $(shell docker info > /dev/null 2>&1 && which docker))
## NAMESPACE: Namespace for `make deploy`
NAMESPACE=korrel8r
## CONFIG: Configuration file for `make run`
CONFIG?=etc/korrel8r/openshift-route.yaml

# Name of image.
IMG?=$(REGISTRY)/$(REGISTRY_ORG)/korrel8r
IMAGE=$(IMG):$(VERSION)

include .bingo/Variables.mk	# Versioned tools

BIN ?= _bin
export PATH := $(abspath $(BIN)):$(PATH)
export GOCOVERDIR := $(abspath _cover)

$(BIN):
	mkdir -p $(BIN)

# Generated files
VERSION_TXT=internal/pkg/build/version.txt
SWAGGER_SPEC=pkg/rest/docs/swagger.json
GEN_SRC=$(VERSION_TXT) $(SWAGGER_SPEC) pkg/config/zz_generated.deepcopy.go
GEN_DOC=doc/gen/domains.adoc doc/gen/rest_api.adoc doc/gen/cmd

all: lint build test _site image-build ## Build and test everything locally. Recommended before pushing.

KORREL8R=./cmd/korrel8r/korrel8r
build: $(KORREL8R)
$(KORREL8R): $(GEN_SRC) $(shell find -name *.go) $(BIN)
	go build -cover -o $@ ./cmd/korrel8r

clean: ## Remove generated files, including checked-in files.
	rm -rf bin _site $(GEN_SRC) doc/gen tmp $(BIN) $(KORREL8R)

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

pkg/config/zz_generated.deepcopy.go:  $(filter-out pkg/config/zz_generated.deepcopy.go,$(wildcard pkg/config/*.go)) $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object paths=./pkg/config/...

$(SWAGGER_SPEC): $(wildcard pkg/rest/*.go) $(SWAG)
	@mkdir -p $(dir $@)
	$(SWAG) init -q -g pkg/rest/operations.go -o $(dir $@)
	$(SWAG) fmt pkg/rest
	@touch $@

SHELLCHECK:= $(BIN)/shellcheck
$(SHELLCHECK):
	@mkdir -p $(dir $@)
	./hack/install-shellcheck.sh $(BIN) 0.10.0

lint: $(GEN_SRC) $(GOLANGCI_LINT) $(SHFMT) $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	hack/copyright.sh
	go mod tidy
	$(GOLANGCI_LINT) run --fix
	$(SHFMT) -l -w ./**/*.sh
	$(SHELLCHECK) -x -S style hack/*.sh

.PHONY: test

test: $(GOCOVERDIR)		## Run all tests, requires a cluster.
	go test -timeout=1m -cover -race ./...
	@echo -e "\\n# Accumulated coverage from main_test"
	go tool covdata percent -i $(GOCOVERDIR)

$(GOCOVERDIR):
	@mkdir -p $@

run: $(KORREL8R) ## Run `korrel8r web` for debugging.
	$(KORREL8R) web -c $(CONFIG)

image-build:  $(GEN_SRC) ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) push -q $(IMAGE)

WAIT_DEPLOYMENT=hack/wait.sh rollout $(NAMESPACE) deployment.apps/korrel8r
DEPLOY_ROUTE=kubectl apply -k config/route -n $(NAMESPACE) || echo "skipping route" # Non-openshift cluster

kustomize-edit: $(KUSTOMIZE)
	cd config;								\
	$(KUSTOMIZE) edit set image "quay.io/korrel8r/korrel8r=$(IMAGE)";	\
	$(KUSTOMIZE) edit set namespace "$(NAMESPACE)"

deploy: image kustomize-edit	## Deploy to current cluster using kustomize.
	kubectl apply -k config
	$(DEPLOY_ROUTE)
	$(WAIT_DEPLOYMENT)

undeploy:			# Delete resources created by `make deploy`
	@kubectl delete -k config/route || true
	@kubectl delete -k config || true

## Documentation

ASCIIDOCTOR:=$(BIN)/asciidoctor
$(ASCIIDOCTOR):
	@mkdir -p $(dir $@)
	gem install asciidoctor --user-install --bindir $(BIN)

# From github.com:darshandsoni/asciidoctor-skins.git
CSS?=adoc-readthedocs.css
ADOC_FLAGS=-a allow-uri-read -a stylesdir=$(shell pwd)/doc/css -a stylesheet=$(CSS)  -a revnumber=$(VERSION) -a revdate=$(shell date -I)

# _site is published to github pages by .github/workflows/asciidoctor-ghpages.yml.
_site: doc $(shell find doc/images etc/korrel8r) $(ASCIIDOCTOR) $(MAKEFILE_LIST) ## Generate the website HTML.
	@mkdir -p $@
	@cp -r doc/images etc/korrel8r $@
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D_site doc/index.adoc
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D_site/gen/cmd doc/gen/cmd/*.adoc
	$(and $(shell type -p linkchecker),linkchecker --no-warnings --check-extern --ignore-url 'https?://localhost[:/].*' _site)
	@touch $@

_site/man: $(KORREL8R)	## Generated man pages.
	@mkdir -p $@
	$(KORREL8R) doc man $@
	@touch $@

doc: $(GEN_DOC)
	touch $@

doc/gen/domains.adoc: $(shell find cmd/korrel8r-doc internal pkg -name '*.go') $(GEN_SRC)
	@mkdir -p $(dir $@)
	go run ./cmd/korrel8r-doc pkg/domains/* > $@

doc/gen/rest_api.adoc: $(SWAGGER_SPEC) $(shell find etc/swagger) $(SWAGGER)
	@mkdir -p $(dir $@)
	$(SWAGGER) -q generate markdown -T etc/swagger -f $(SWAGGER_SPEC) --output $@

KRAMDOC:=$(BIN)/kramdoc
$(KRAMDOC):
	@mkdir -p $(dir $@)
	gem install kramdown-asciidoc --user-install --bindir $(BIN)

doc/gen/cmd: $(KORREL8R) $(KORREL8RCLI) $(KRAMDOC) ## Generated command documentation
	@mkdir -p $@
	$(KORREL8R) doc markdown $@
	$(KORREL8RCLI) doc markdown $@
	hack/md-to-adoc.sh $(KRAMDOC) $@/*.md
	@touch $@

pre-release:	## Prepare for a release. Push results before `make release`
	@[ "$(origin REGISTRY_ORG)" = "command line" ] || { echo "REGISTRY_ORG must be set on the command line for a release."; exit 1; }
	$(MAKE) all image
	@echo Ready to release $(VERSION), images at $(REGISTRY_ORG)

release:			## Push images and release tags for a release.
	$(MAKE) clean
	$(MAKE) pre-release
	hack/tag-release.sh $(VERSION)
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"
	@echo Released $(VERSION), images at $(REGISTRY_ORG)

BINGO=$(GOBIN)/bingo
$(BINGO): # Bootstrap bingo
	go install github.com/bwplotka/bingo@v0.9.0

tools: $(BINGO) $(ASCIIDOCTOR) $(KRAMDOC) $(SHELLCHECK) ## Download all tools needed for development
	$(BINGO) get
