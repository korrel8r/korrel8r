# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release, use -dev for development pre-release versions.
VERSION?=0.8.2-dev
## REGISTRY_BASE: Image registry base, for example quay.io/somebody
REGISTRY_BASE?=$(error REGISTRY_BASE must be set to push images)
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell type -p podman || type -p docker),$(error No podman or docker))
## NAMESPACE: Namespace for `make deploy`
NAMESPACE=korrel8r
## CONFIG: Configuration file for `make run`
CONFIG?=etc/korrel8r/openshift-route.yaml

# Name of image.
IMG?=$(REGISTRY_BASE)/korrel8r
IMAGE=$(IMG):$(VERSION)

include .bingo/Variables.mk	# Versioned tools

BIN ?= _bin
export PATH := $(abspath $(BIN)):$(PATH)
export GOCOVERDIR := $(abspath _cover)

$(BIN):
	mkdir -p $(BIN)
_cover:
	mkdir -p _cover

# Generated files
VERSION_TXT=internal/pkg/build/version.txt
OPENAPI_SPEC=doc/korrel8r-openapi.yaml

generate: $(VERSION_TXT) pkg/rest/oapi-codegen.go _cover

all: lint test _site image-build ## Build and test everything locally. Recommended before pushing.

build: generate $(BIN)				## Build korrel8r executable.
	go build -o $(BIN)/korrel8r ./cmd/korrel8r

install: generate							## Build and install korrel8r with go install.
	go install ./cmd/korrel8r

run: generate									## Run `korrel8r web` for debugging.
	go run ./cmd/korrel8r web -c $(CONFIG) $(KORREL8R_FLAGS)

clean: ## Remove generated files, including checked-in files.
	rm -rf _site generate doc/gen tmp $(BIN) $(GOCOVERDIR)

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

pkg/rest/oapi-codegen.go: $(OPENAPI_SPEC) $(OAPI_CODEGEN)
	$(OAPI_CODEGEN) -generate types,gin,spec -package rest -o $@ $<

SHELLCHECK:= $(BIN)/shellcheck
$(SHELLCHECK):
	@mkdir -p $(dir $@)
	./hack/install-shellcheck.sh $(BIN) 0.10.0

lint: generate $(GOLANGCI_LINT) $(SHFMT) $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	hack/copyright.sh
	go mod tidy
	$(GOLANGCI_LINT) run --fix
	$(SHFMT) -l -w ./**/*.sh
	$(SHELLCHECK) -x -S style hack/*.sh

.PHONY: test
test: generate		## Run all tests, no cache. Requires an openshift cluster.
	go test -fullpath -race ./...

test-no-cluster: generate	## Run all tests that don't require an openshift cluster.
	go test -fullpath -race -skip '.*/Openshift' ./...

cover:  $(GOCOVERDIR) ## Run tests with accumulated coverage stats in _cover.
	@echo == Individual package test coverage.
	go test -fullpath -cover ./... -test.gocoverdir=$(GOCOVERDIR)
	@echo
	@echo == Aggregate coverage across all tests.
	go tool covdata percent -i $(GOCOVERDIR)

bench: generate	$(BENCHSTAT)	## Run all benchmarks.
	go test -fullpath -bench=. -run=NONE ./... > _bench-$(shell date +%s).txt ; $(BENCHSTAT) _bench-*.txt

$(GOCOVERDIR):
	@mkdir -p $@

image-build:  generate ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) push -q $(IMAGE)

image-latest: image ## Build and push image with 'latest' alias
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"

WAIT_DEPLOYMENT=hack/wait.sh rollout $(NAMESPACE) deployment.apps/korrel8r
DEPLOY_ROUTE=kubectl apply -k config/route -n $(NAMESPACE) || echo "skipping route" # Non-openshift cluster

kustomize-edit: $(KUSTOMIZE)
	cd config;								\
	$(KUSTOMIZE) edit set image "$(REGISTRY_BASE)/korrel8r=$(IMAGE)";	\
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
ADOC_FLAGS=-v -a allow-uri-read -a stylesdir=$(shell pwd)/doc/css -a stylesheet=$(CSS)  -a revnumber=$(VERSION) -a revdate=$(shell date -I)
LINKCHECKER?=$(or $(shell type -p linkchecker),$(warning linkchecker not found: skipping link checks))
LINKCHECK_FLAGS=--no-warnings --check-extern --ignore-url='//(localhost|[^:/]*\.example)([:/].*)?$$'

# _site is published to github pages by .github/workflows/asciidoctor-ghpages.yml.
_site: doc $(shell find doc etc/korrel8r -name gen -prune -o -print) $(ASCIIDOCTOR) ## Generate the website HTML.
	git submodule init
	git submodule update --force
	@mkdir -p $@/doc/images
	cp -r doc/images etc/korrel8r $@
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D$@ doc/README.adoc -o index.html
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D$@/gen/cmd doc/gen/cmd/*.adoc
	$(and $(LINKCHECKER),$(LINKCHECKER) $(LINKCHECK_FLAGS) $@)
	@touch $@

_site/man: $(shell find ./cmd)	## Generated man pages.
	@mkdir -p $@
	go run ./cmd/korrel8r doc man $@
	@touch $@

doc: doc/gen/domains.adoc doc/gen/rest_api.adoc doc/gen/cmd
	@touch $@

DOMAINS=$(shell echo pkg/domains/*/ | xargs basename -a)
DOMAIN_PKGS=$(foreach D,$(DOMAINS),github.com/korrel8r/korrel8r/pkg/domains/$(D))
doc/gen/domains.adoc: $(shell find pkg/domains internal/cmd/domain-adoc internal/pkg/asciidoc) generate
	@mkdir -p $(dir $@)
	go run ./internal/cmd/domain-adoc $(DOMAIN_PKGS) > $@

OPENAPI_CFG=doc/openapi-asciidoc.yaml
doc/gen/rest_api.adoc: $(OPENAPI_SPEC) $(OPENAPI_GEN) $(OPENAPI_CFG)
	@mkdir -p $(dir $@)
	$(IMGTOOL) run --rm -v $(shell pwd):/app docker.io/openapitools/openapi-generator-cli \
		generate -g asciidoc -c app/$(OPENAPI_CFG) -i app/$< -o app/$@.dir
	@mv -f $@.dir/index.adoc $@
	@rm -rf $@.dir

KRAMDOC:=$(BIN)/kramdoc
$(KRAMDOC):
	@mkdir -p $(dir $@)
	gem install kramdown-asciidoc --user-install --bindir $(BIN)

doc/gen/cmd: $(shell find ./cmd/korrel8r) $(KORREL8RCLI) $(KRAMDOC) ## Generated command documentation
	@mkdir -p $@
	unset KORREL8R_CONFIG; go run ./cmd/korrel8r doc markdown $@
	$(KORREL8RCLI) doc markdown $@
	hack/md-to-adoc.sh $(KRAMDOC) $@/*.md
	@touch $@

pre-release:	## Prepare for a release. Push results before `make release`
	$(MAKE) all image kustomize-edit
	@echo Ready to release $(VERSION) to $(REGISTRY_BASE)

release:			## Push images and release tags for a release.
	$(MAKE) clean
	$(MAKE) pre-release
	hack/tag-release.sh $(VERSION)
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"
	@echo Released $(VERSION) to $(REGISTRY_BASE)

BINGO=$(GOBIN)/bingo
$(BINGO): # Bootstrap bingo
	go install github.com/bwplotka/bingo@v0.9.0

tools: $(BINGO) $(ASCIIDOCTOR) $(KRAMDOC) $(SHELLCHECK) ## Download all tools needed for development
	$(BINGO) get

