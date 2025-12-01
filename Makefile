# Makefile is self-documenting, comments starting with '##' are extracted as help text.
# Variables: ## line comment before the variable..
# Targets: ## inline comment after the target.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release, use -dev for development pre-release versions.
VERSION?=0.8.5-dev
## REGISTRY_BASE: Image registry base, for example quay.io/somebody
REGISTRY_BASE?=$(error REGISTRY_BASE must be set to push images)
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell which podman 2>/dev/null),$(shell which docker 2>/dev/null),$(error Cannot find podman or docker in PATH))
## NAMESPACE: Namespace for `make deploy`
NAMESPACE=korrel8r

# Name of image.
IMG?=$(REGISTRY_BASE)/korrel8r
IMAGE=$(IMG):$(VERSION)

include .bingo/Variables.mk	# Versioned tools

BIN ?= _bin
export PATH := $(abspath $(BIN)):$(PATH)
export GOCOVERDIR := $(abspath _cover)
$(shell mkdir -p $(GOCOVERDIR))

# Generated files
VERSION_TXT=internal/pkg/build/version.txt
OPENAPI_SPEC=doc/korrel8r-openapi.yaml
GEN_OPENAPI_GO=pkg/rest/gen-openapi.go
GENERATED=$(VERSION_TXT) $(GEN_OPENAPI_GO)

generate:  $(GENERATED)

all: lint test _site image-build ## Build and test everything locally. Recommended before commit.

build: $(GENERATED) $(BIN)				## Build korrel8r executable.
	go build -o $(BIN)/korrel8r ./cmd/korrel8r

install: $(GENERATED)							## Build and install korrel8r with go install.
	go install ./cmd/korrel8r

clean: ## Remove generated files, including checked-in files.
	rm -rf _site $(GENERATED) doc/gen tmp $(GEN_OPENAPI_GO) $(BIN) $(GOCOVERDIR)

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

$(BIN):
	mkdir -p $(BIN)

$(GEN_OPENAPI_GO): $(OPENAPI_SPEC) $(OAPI_CODEGEN)
	$(OAPI_CODEGEN) -generate types,gin,spec -package rest -o $@ $<

SHELLCHECK:= $(BIN)/shellcheck
$(SHELLCHECK): $(BIN)
	./hack/install-shellcheck.sh $(BIN) 0.10.0

lint: $(GENERATED) $(GOLANGCI_LINT) $(SHFMT) $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	hack/copyright.sh
	go mod tidy
	$(GOLANGCI_LINT) run --fix
	$(SHFMT) -l -w ./**/*.sh
	$(SHELLCHECK) -x -S style hack/*.sh

.PHONY: test
test: $(GENERATED)		## Run all tests, no cache. Requires an openshift cluster.
	go test -fullpath -race ./...

test-no-cluster: $(GENERATED)	## Run all tests that don't require an openshift cluster.
	go test -fullpath -race  -skip='Cluster|/Cluster' ./...

test-clean: ## Remove test namespaces from the cluster
	kubectl delete ns -l test=korrel8r

.PHONY: cover
cover:  ## Run tests with accumulated coverage stats in _cover.
	@rm -rf  $(GOCOVERDIR) ; mkdir -p $(GOCOVERDIR)
	@echo == Individual package test coverage.
	go test -fullpath -cover ./... -test.gocoverdir=$(GOCOVERDIR)
	@echo
	@echo == Aggregate coverage across all tests.
	go tool covdata percent -i $(GOCOVERDIR)

bench: $(GENERATED)	$(BENCHSTAT)	## Run all benchmarks.
	go test -fullpath -bench=. -run=NONE ./... > _bench-$(shell date +%s).txt ; $(BENCHSTAT) _bench-*.txt

image-build:  $(GENERATED) ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image.
	$(IMGTOOL) push -q $(IMAGE)

image-latest: image ## Build and push image with 'latest' alias
	$(IMGTOOL) push -q $(IMAGE) $(IMG):latest

WAIT_DEPLOYMENT=hack/wait.sh rollout $(NAMESPACE) deployment.apps/korrel8r
DEPLOY_ROUTE=kubectl apply -k config/route -n $(NAMESPACE) || echo "skipping route" # Non-openshift cluster

kustomize-edit: $(KUSTOMIZE)
	cd config;								\
	$(KUSTOMIZE) edit set image "$(REGISTRY_BASE)/korrel8r=$(IMAGE)";	\
	$(KUSTOMIZE) edit set namespace "$(NAMESPACE)"

deploy: image kustomize-edit ## Deploy to current cluster using kustomize.
	kubectl apply -k config
	$(DEPLOY_ROUTE)
	$(WAIT_DEPLOYMENT)

undeploy:			## Delete resources created by `make deploy`
	@kubectl delete -k config/route || true
	@kubectl delete -k config || true

ASCIIDOCTOR:=$(BIN)/asciidoctor
$(ASCIIDOCTOR): $(BIN)
	gem install asciidoctor --user-install --bindir $(BIN)

# From github.com:darshandsoni/asciidoctor-skins.git
CSS?=adoc-readthedocs.css
ADOC_FLAGS=-v -a allow-uri-read -a stylesdir=$(shell pwd)/doc/css -a stylesheet=$(CSS)  -a revnumber=$(VERSION) -a revdate=$(shell date -I)
LINKCHECKER?=$(or $(shell type -p linkchecker),$(warning linkchecker not found: skipping link checks))
LINKCHECK_FLAGS=--no-warnings --check-extern --ignore-url='//(localhost|[^:/]*\.example)([:/].*)?$$'

# _site is published to github pages by .github/workflows/asciidoctor-ghpages.yml.
_site: doc $(shell find etc/korrel8r -name gen -prune -o -print) $(ASCIIDOCTOR) ## Generate the website HTML.
	git submodule init
	git submodule update --force
	@mkdir -p $@/doc/images
	cp -r doc/images etc/korrel8r $@
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D$@ doc/user-guide.adoc -o index.html
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D$@/gen/cmd doc/gen/cmd/*.adoc
	$(and $(LINKCHECKER),$(LINKCHECKER) $(LINKCHECK_FLAGS) $@)
	@touch $@

_site/man: $(shell find ./cmd)	## Generated man pages.
	@mkdir -p $@
	go run ./cmd/korrel8r doc man $@
	@touch $@

doc: doc/gen/domains.adoc doc/gen/rest_api.adoc doc/gen/cmd
	touch $@

doc/gen/domains.adoc: $(wildcard pkg/domains/*/*.adoc)
	@mkdir -p $(dir $@)
	rm -f $@; for D in $^; do { echo; echo "include::../../$$D[]"; echo; } >>$@; done

doc/gen/rest_api.adoc: $(OPENAPI_SPEC) $(OPENAPI_GEN)
	@mkdir -p $(dir $@)
	$(IMGTOOL) run --rm -v $(CURDIR):/app:z docker.io/openapitools/openapi-generator-cli \
		generate -g asciidoc -c /app/doc/openapi-asciidoc.yaml -i /app/$< -o /app/$@.dir
	@mv -f $@.dir/index.adoc $@
	@rm -rf $@.dir

KRAMDOC:=$(BIN)/kramdoc
$(KRAMDOC):
	@mkdir -p $(dir $@)
	gem install kramdown-asciidoc --user-install --bindir $(BIN)

doc/gen/cmd: $(GENERATED) $(shell find ./cmd/korrel8r) $(KORREL8RCLI) $(KRAMDOC) ## Generated command documentation
	@mkdir -p $@
	unset KORREL8R_CONFIG; go run ./cmd/korrel8r doc markdown $@
	$(KORREL8RCLI) doc markdown $@
	hack/md-to-adoc.sh $(KRAMDOC) $@/*.md
	@touch $@

# See doc/RELEASE.adoc
pre-release:
	$(MAKE) all image kustomize-edit
	@echo Ready to release $(VERSION) to $(REGISTRY_BASE)

# See doc/RELEASE.adoc
release:
	$(MAKE) clean
	$(MAKE) pre-release
	hack/tag-release.sh $(VERSION)
	$(MAKE) image-latest
	@echo Released $(VERSION) to $(REGISTRY_BASE)

BINGO=$(GOBIN)/bingo
$(BINGO): # Bootstrap bingo
	go install github.com/bwplotka/bingo@v0.9.0

# Force download of all tools needed for development
tools: $(BINGO) $(ASCIIDOCTOR) $(KRAMDOC) $(SHELLCHECK)
	$(BINGO) get

DEVSPACE_IMAGE?="$(REGISTRY_BASE)/korrel8r:devspace"
devspace-image:									## Rebuild the devspace base image
	$(IMGTOOL) build --tag=$(DEVSPACE_IMAGE) -f devspace.Containerfile
	$(IMGTOOL) push $(DEVSPACE_IMAGE)
