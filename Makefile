# Makefile is self-documenting, comments starting with '##' are extracted as help text.
# Variables: ## line comment before the variable..
# Targets: ## inline comment after the target.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -hE '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -hE '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release, use -dev for development pre-release versions.
VERSION?=0.11.2-dev
## REGISTRY_BASE: Image registry base, for example quay.io/somebody
REGISTRY_BASE?=$(error REGISTRY_BASE must be set to push images)
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell which podman 2>/dev/null),$(shell which docker 2>/dev/null),$(error Cannot find podman or docker in PATH))
## NAMESPACE: Namespace for `make deploy`
NAMESPACE=korrel8r

# Name of image.
IMG?=$(REGISTRY_BASE)/korrel8r
IMAGE=$(IMG):$(VERSION)

BIN ?= _bin
export PATH := $(abspath $(BIN)):$(PATH)
export GOCOVERDIR := $(abspath _cover)
$(shell mkdir -p $(GOCOVERDIR))

# Sources for generated files
OPENAPI_SPEC=korrel8r-openapi.yaml
DOMAINS=$(patsubst pkg/domains/%/doc.go,%,$(wildcard pkg/domains/*/doc.go))

# Generated files required to build the Go code.
VERSION_TXT=internal/pkg/build/version.txt
GEN_OPENAPI_API=pkg/api/gen-openapi.go
GEN_OPENAPI_IMPL=pkg/rest/gen-openapi.go
GEN_DOMAIN_DOC=$(patsubst %.go,%.md,$(wildcard pkg/domains/*/doc.go))

GENERATED=$(VERSION_TXT) $(GEN_OPENAPI_IMPL) $(GEN_OPENAPI_API) $(GEN_DOMAIN_DOC)

all: test doc image-build ## Build and test everything locally. Recommended before commit.

generate:  $(GENERATED)
	hack/copyright.sh

$(OPENAPI_SPEC): $(VERSION_TXT) # Stamp x-korrel8r-version in the OpenAPI spec.
	@sed -i -e '/x-korrel8r-version/d' -e '/^  version: /a\  x-korrel8r-version: "$(VERSION)"' $@

build: lint $(BIN)				## Build korrel8r executable.
	go build -o $(BIN)/korrel8r ./cmd/korrel8r

install: lint							## Build and install korrel8r with go install.
	go install ./cmd/korrel8r

# Append more cleanup files with +=
CLEANFILES+=$(GENERATED) $(BIN) $(GOCOVERDIR)
clean: ## Remove generated files, including checked-in files.
	-rm -rf $(CLEANFILES)

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

$(BIN):
	mkdir -p $(BIN)

$(GEN_OPENAPI_API): $(OPENAPI_SPEC)
	go tool oapi-codegen -generate types,spec -package api -o $@ $<

$(GEN_OPENAPI_IMPL): $(OPENAPI_SPEC)
	go tool oapi-codegen -generate gin -package rest -import-mapping "$<:github.com/korrel8r/korrel8r/pkg/api" -alias-types -o $@ $<

# Generate the pkg/domains/*/doc.md files that are embedded in the korrel8r executable.
%/doc.md: %/doc.go $(shell find doc/gomarkdoc)
	@mkdir -p $(dir $@)
	go tool gomarkdoc ./$(dir $@) --template-file file=doc/gomarkdoc/file.gotxt | sed -E 's/[Pp]ackage *[a-zA-Z0-9]+ *(is a)? *(korrel8r)? *(domain)? *(for)? *//' > $@

SHELLCHECK:= $(BIN)/shellcheck
$(SHELLCHECK): $(BIN)
	./hack/install-shellcheck.sh $(BIN) 0.10.0

ifndef NOLINT
lint: generate $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	go mod tidy
	go tool golangci-lint run --fix
	go tool shfmt -l -w ./**/*.sh
	$(SHELLCHECK) -x -S style hack/*.sh
else
lint: ## Linting skipped (NOLINT is set).
endif

.PHONY: test
test: lint											## Run all tests, no cache. Requires an openshift cluster.
	go test -fullpath -race ./...

test-no-cluster: lint	## Run all tests that don't require an openshift cluster.
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

bench: generate	## Run all benchmarks.
	go test -fullpath -bench=. -run=NONE ./... > _bench-$(shell date -I).txt
	go tool benchstat _bench-*.txt

image-build: lint kustomize-edit ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image.
	$(IMGTOOL) push -q $(IMAGE)

image-latest: image ## Build and push image with 'latest' alias
	$(IMGTOOL) push -q $(IMAGE) $(IMG):latest

WAIT_DEPLOYMENT=hack/wait.sh rollout $(NAMESPACE) deployment.apps/korrel8r
DEPLOY_ROUTE=kubectl apply -k config/route -n $(NAMESPACE) || echo "skipping route" # Non-openshift cluster

kustomize-edit:
	cd config; \
	go tool kustomize edit set image "$(REGISTRY_BASE)/korrel8r=$(IMAGE)"; \
	go tool kustomize edit set namespace "$(NAMESPACE)"

deploy: image ## Deploy to current cluster using kustomize.
	kubectl apply -k config
	$(DEPLOY_ROUTE)
	$(WAIT_DEPLOYMENT)

undeploy:			## Delete resources created by `make deploy`
	@kubectl delete -k config/route || true
	@kubectl delete -k config || true

# See RELEASE.md
pre-release:
	$(MAKE) all image kustomize-edit
	@echo Ready to release $(VERSION) to $(REGISTRY_BASE)

# See RELEASE.md
release:
	$(MAKE) clean
	$(MAKE) pre-release
	hack/tag-release.sh $(VERSION)
	$(MAKE) image-latest
	@echo Released $(VERSION) to $(REGISTRY_BASE)

DEVSPACE_IMAGE?="$(REGISTRY_BASE)/korrel8r:devspace"
devspace-image:	kustomize-edit ## Rebuild the devspace base image
	$(IMGTOOL) build --tag=$(DEVSPACE_IMAGE) -f devspace.Containerfile
	$(IMGTOOL) push $(DEVSPACE_IMAGE)

## Documentation rules

doc: doc/public
	$(MAKE) check-links

.PHONY: preview
preview: doc/public
	@rm -rf $<
	go tool hugo server --source doc --baseURL http://localhost:1313 --bind 0.0.0.0
	@touch $<

.PHONY: check-links
check-links: doc/public ## Check for broken internal links in the generated site.
	hack/check-links.sh doc/public "^/client/"

# Pre-pends front matter to file: --- title: description:
FRONT=./hack/front-matter.sh

DOC_PUBLIC+=$(foreach D,$(DOMAINS), doc/content/docs/reference/domains/$(D).md)
doc/content/docs/reference/domains/%.md: pkg/domains/%/doc.md generate
	@mkdir -p $(dir $@)
	@cp $< $@
	@$(FRONT) $@ 'title: $(basename $(notdir $@))' "description: $$(sed -n '/^[^#]/{/./p;q}' $@)"


DOC_PUBLIC+=doc/content/docs/reference/rest/index.md
CLEANFILES+=doc/content/docs/reference/rest
doc/content/docs/reference/rest/index.md: $(OPENAPI_SPEC) $(MAKEFILE_LIST)
	@mkdir -p $(dir $@)
	go tool openapi-markdown -o $@ -title "REST API" -description "HTTP API reference" $<
	@sed -i 's/^# REST API$$//'	$@				# Remove redundant header
	@perl -pi -e 'if (/^### ((?:PUT|GET|POST|DELETE|PATCH) .+)$$/) { my $$t = $$1; (my $$a = lc $$t) =~ s/[^a-z0-9]//g; $$_ = "### $$t {#$$a}\n"; }' $@ # Fix anchors
	@$(FRONT) $@ 'title: REST API' 'description: HTTP API reference'

DOC_PUBLIC+=doc/content/docs/reference/cmd/_index.md
CLEANFILES+=doc/content/docs/reference/cmd
doc/content/docs/reference/cmd/_index.md: generate  $(shell find cmd pkg)
	@mkdir -p $(dir $@)
	unset KORREL8R_CONFIG; go run ./cmd/korrel8r doc markdown $(dir $@)
	@mv $(dir $@)korrel8r.md $@
	@$(FRONT) $@ "title: Korrel8r Command" "description: Command line interface"
	@sed -i '/###### Auto generated by/d' $@
	@for f in $(dir $@)korrel8r_*.md; do \
		$(FRONT) "$$f" "title: $$(basename "$$f" .md | sed 's/_/ /g')"; \
		sed -i '/^### SEE ALSO/,$$d' "$$f"; \
	done
	@touch $@

doc/public: $(DOC_PUBLIC) $(shell find doc/* -name public -prune -o -print)
	go tool hugo --source doc
	@touch $@
CLEANFILES+=doc/public
CLEANFILES+=$(DOC_PUBLIC)
