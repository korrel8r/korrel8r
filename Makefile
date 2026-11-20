# Makefile is self-documenting, comments starting with '##' are extracted as help text.
# Variables: ## line comment before the variable..
# Targets: ## inline comment after the target.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -hE '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -hE '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release, use -dev for development pre-release versions.
VERSION?=0.13.9-dev
## REGISTRY_BASE: Image registry base, defaults to quay.io/korrel8r.
REGISTRY_BASE?=quay.io/korrel8r
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell which podman 2>/dev/null),$(shell which docker 2>/dev/null),$(error Cannot find podman or docker in PATH))
## NAMESPACE: Namespace for `make deploy`
NAMESPACE=korrel8r

# Name of image.
IMG?=$(REGISTRY_BASE)/korrel8r
IMAGE=$(IMG):$(VERSION)

BIN ?= _bin
export GOCOVERDIR := $(abspath _cover)
$(shell mkdir -p $(GOCOVERDIR))

# Tools from the tools module, included in go.work.
GO_TOOL = go tool
TOOL_DEPS = tools/go.mod tools/go.sum
$(BIN):
	@mkdir -p $@

# Main and sub-modules
GO_MODULES=. ./pkg/mcp ./pkg/api ./tools

# Sources for generated files
OPENAPI_SPEC=korrel8r-openapi.yaml
DOMAINS=$(patsubst pkg/domains/%/doc.go,%,$(wildcard pkg/domains/*/doc.go))

# Generated source files required to build the Go code.
VERSION_TXT=internal/pkg/build/version.txt
GEN_OPENAPI_API=pkg/api/gen-openapi.go
GEN_OPENAPI_IMPL=pkg/rest/gen-openapi.go
# These markdown files are embedded in the executable.
GEN_DOMAIN_DOC=$(patsubst %.go,%.md,$(wildcard pkg/domains/*/doc.go))
# Compiled quicktemplate rules, see pkg/rules/quickrules.
GEN_QUICKRULES=pkg/rules/quickrules/applyfuncs.go

GENERATED=$(VERSION_TXT) $(GEN_OPENAPI_IMPL) $(GEN_OPENAPI_API) $(GEN_DOMAIN_DOC) $(GEN_QUICKRULES)

all: test doc vuln image-build ## Build and test everything locally. Recommended before commit.
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

$(GEN_OPENAPI_API): $(OPENAPI_SPEC) $(TOOL_DEPS)
	$(GO_TOOL) oapi-codegen -generate types,spec -package api -o $@ $<

$(GEN_OPENAPI_IMPL): $(OPENAPI_SPEC) $(TOOL_DEPS)
	$(GO_TOOL) oapi-codegen -generate gin -package rest -import-mapping "$<:github.com/korrel8r/korrel8r/pkg/api" -alias-types -o $@ $<

# Compile quicktemplate files to go.
$(GEN_QUICKRULES): $(wildcard $(dir $(GEN_QUICKRULES))*.qtpl) $(TOOL_DEPS)
	go generate ./$(dir $@)
	hack/gen-applyfuncs.sh $(dir $@) | gofmt > $@

# Generate the pkg/domains/*/doc.md files that are embedded in the korrel8r executable.
%/doc.md: %/doc.go $(shell find doc/gomarkdoc) $(TOOL_DEPS)
	@mkdir -p $(dir $@)
	@$(GO_TOOL) gomarkdoc ./$(dir $@) --template-file file=doc/gomarkdoc/file.gotxt | sed -E 's/[Pp]ackage *[a-zA-Z0-9]+ *(is a)? *(korrel8r)? *(domain)? *(for)? *//' > $@

SHELLCHECK:= $(BIN)/shellcheck
$(SHELLCHECK):
	@mkdir -p $(BIN)
	./hack/install-shellcheck.sh $(BIN) 0.10.0

.PHONY: check-tools
check-tools: ## Build all pinned Go tools to catch incompatible transitive dependencies.
	go -C tools build tool

.PHONY: go-mod-tidy
# Generated Go files must exist before tidy, especially after make clean.
go-mod-tidy: generate ## Tidy dependencies in the main and submodules.
	@set -e; for d in $(GO_MODULES); do go -C "$$d" mod tidy; done

ifndef NOLINT
lint: go-mod-tidy check-tools generate $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	$(GO_TOOL) golangci-lint run --fix
	$(GO_TOOL) shfmt -l -w ./**/*.sh
	$(SHELLCHECK) -x -S style hack/*.sh
else
lint: ## Linting skipped (NOLINT is set).
endif

# Setting NO_CLUSTER=1 skips cluster tests
TEST_FLAGS?=$(and $(NO_CLUSTER),-skip='_cluster(-fm)?$$|/_cluster(-fm)?$$')

.PHONY: test
test: lint							 ## Run all tests, requires cluster. Set NO_CLUSTER=1 to skip cluster tests.
	go test -fullpath -race ./... $(TEST_FLAGS)
	go -C pkg/api test -fullpath -race ./... $(TEST_FLAGS)
	go -C pkg/mcp test -fullpath -race ./... $(TEST_FLAGS)

.PHONY: vuln
vuln: ## Check Go dependencies for known vulnerabilities.
	go version
	$(GO_TOOL) govulncheck $(VULN_FLAGS) ./...

test-clean: ## Remove test namespaces from the cluster
	kubectl delete ns -l test=korrel8r

## TEST_SETUP_REPO: Repository of cluster test-setup scripts, see https://github.com/korrel8r/config
TEST_SETUP_REPO?=https://github.com/korrel8r/config.git
## TEST_SETUP_REF: Branch, tag or commit of TEST_SETUP_REPO to use.
TEST_SETUP_REF?=main
## TEST_SETUP_DIR: Local checkout of TEST_SETUP_REPO, set to an existing clone to work from it directly.
TEST_SETUP_DIR?=_test-setup

$(TEST_SETUP_DIR):
	git clone --filter=blob:none $(TEST_SETUP_REPO) $@
	git -C $@ checkout $(TEST_SETUP_REF)

.PHONY: test-setup-update
test-setup-update: $(TEST_SETUP_DIR)  ## Update the test-setup checkout to TEST_SETUP_REF.
	git -C $(TEST_SETUP_DIR) fetch origin
	git -C $(TEST_SETUP_DIR) checkout $(TEST_SETUP_REF)
	-git -C $(TEST_SETUP_DIR) pull --ff-only

.PHONY: test-setup
test-setup: test-setup-update  ## Install observability signals on the current OpenShift cluster, for `make test`.
	$(MAKE) -C $(TEST_SETUP_DIR)/openshift all -j5

.PHONY: test-setup-clean
test-setup-clean: $(TEST_SETUP_DIR)  ## Remove observability components installed by `make test-setup`.
	$(MAKE) -C $(TEST_SETUP_DIR)/openshift clean-cluster -j5

.PHONY: cover
cover:  ## Run tests with accumulated coverage stats in _cover.
	@rm -rf  $(GOCOVERDIR) ; mkdir -p $(GOCOVERDIR)
	@echo ==== Individual package test coverage ====
	go test -fullpath -cover $(TEST_FLAGS) ./... -test.gocoverdir=$(GOCOVERDIR)
	@echo
	@echo ==== Aggregate coverage across all tests ====
	go tool covdata percent -i $(GOCOVERDIR)

DESCRIBE=$(or $(shell git describe --all --dirty | sed 's-^heads/--'),$(error git describe is empty))
BENCH_NAME?=$(DESCRIBE)
BENCH_PKG?=./...
BENCH_COUNT?=6
.PHONY: bench
bench:
	mkdir -p _bench
	go test -v -bench=. -run=- -count=$(BENCH_COUNT) $(BENCH_PKG) | tee _bench/$(BENCH_NAME).bench
	$(GO_TOOL) benchstat _bench/*

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
	$(GO_TOOL) kustomize edit set image "korrel8r=$(IMAGE)"; \
	$(GO_TOOL) kustomize edit set namespace "$(NAMESPACE)"

deploy: image ## Deploy to current cluster using kustomize.
	kubectl apply -k config
	$(DEPLOY_ROUTE)
	$(WAIT_DEPLOYMENT)

undeploy:			## Delete resources created by `make deploy`
	@kubectl delete -k config/route || true
	@kubectl delete -k config || true

pre-release:										## Prepare a release, see RELEASE.md
	$(if $(filter file,$(origin VERSION)),$(error set VERSION explicitly for pre-release))
	$(if $(filter file,$(origin REGISTRY_BASE)),,$(error REGISTRY_BASE must use the Makefile default))
	hack/pre-release.sh $(VERSION)

push-release:									## Publish a release, see RELEASE.md
	$(if $(filter file,$(origin VERSION)),,$(error VERSION must come from the Makefile for push-release))
	$(if $(filter file,$(origin REGISTRY_BASE)),,$(error REGISTRY_BASE must use the Makefile default))
	hack/push-release.sh $(VERSION)

post-release:									## Bump the dev or patch version after release, see RELEASE.md
	$(if $(filter file,$(origin VERSION)),,$(error VERSION must come from the Makefile for post-release))
	$(if $(filter file,$(origin REGISTRY_BASE)),,$(error REGISTRY_BASE must use the Makefile default))
	hack/post-release.sh $(VERSION)

DEVSPACE_TAG?=devspace-$(shell git rev-parse --short HEAD)-$(shell sha256sum go.sum | cut -c1-8)
DEVSPACE_IMAGE?=$(REGISTRY_BASE)/korrel8r:$(DEVSPACE_TAG)
devspace-image:	kustomize-edit ## Rebuild the devspace base image
	$(IMGTOOL) build --tag=$(DEVSPACE_IMAGE) -f devspace.Containerfile .
	$(IMGTOOL) push $(DEVSPACE_IMAGE)
	@mkdir -p .devspace
	@echo $(DEVSPACE_IMAGE) > .devspace/devimage

## Documentation rules

doc: doc/public ## Generate documentation
	$(MAKE) check-links

.PHONY: preview
preview: doc/public ## Preview documentation in browser at http://localhost:1313
	@rm -rf $<
	$(GO_TOOL) hugo server --source doc --baseURL http://localhost:1313 --bind 0.0.0.0
	@touch $<

.PHONY: check-links
check-links: doc/public ## Check for broken internal links in the generated site.
	hack/check-links.sh doc/public "^/client/"

# Pre-pends front matter to file: --- title: description:
FRONT=./hack/front-matter.sh

DOC_PUBLIC+=$(foreach D,$(DOMAINS), doc/content/docs/reference/domains/$(D).md)
doc/content/docs/reference/domains/%.md: pkg/domains/%/doc.md generate $(FRONT)
	@mkdir -p $(dir $@)
	@cp $< $@
	@$(FRONT) $@ 'title: $(basename $(notdir $@))' "description: $$(sed -n '/^[^#]/{/./p;q}' $@)"

DOC_PUBLIC+=doc/content/docs/reference/quickrules.md
doc/content/docs/reference/quickrules.md: pkg/rules/quickrules/doc.md $(FRONT)
	@mkdir -p $(dir $@)
	@mv -f $< $@
	@$(FRONT) $@ 'title: Compiled Rules' "description: Writing compiled quicktemplate rules" 'weight: 20'

DOC_PUBLIC+=doc/content/docs/reference/rest/index.md
CLEANFILES+=doc/content/docs/reference/rest
doc/content/docs/reference/rest/index.md: $(OPENAPI_SPEC) $(MAKEFILE_LIST) $(FRONT) $(TOOL_DEPS)
	@mkdir -p $(dir $@)
	$(GO_TOOL) openapi-markdown -o $@ -title "REST API" -description "HTTP API reference" $<
	@sed -i 's/^# REST API$$//'	$@				# Remove redundant header
	@perl -pi -e 'if (/^### ((?:PUT|GET|POST|DELETE|PATCH) .+)$$/) { my $$t = $$1; (my $$a = lc $$t) =~ s/[^a-z0-9]//g; $$_ = "### $$t {#$$a}\n"; }' $@ # Fix anchors
	@$(FRONT) $@ 'title: REST API' 'description: HTTP API reference' 'weight: 80'

DOC_PUBLIC+=doc/content/docs/reference/cmd/_index.md
CLEANFILES+=doc/content/docs/reference/cmd
doc/content/docs/reference/cmd/_index.md: generate  $(shell find cmd pkg) $(FRONT)
	@mkdir -p $(dir $@)
	unset KORREL8R_CONFIG; go run ./cmd/korrel8r doc markdown $(dir $@)
	@mv $(dir $@)korrel8r.md $@
	@$(FRONT) $@ "title: Korrel8r Command" "description: Command line interface" "weight: 60"
	@sed -i '/###### Auto generated by/d' $@
	@for f in $(dir $@)korrel8r_*.md; do \
		$(FRONT) "$$f" "title: $$(basename "$$f" .md | sed 's/_/ /g')"; \
		sed -i '/^### SEE ALSO/,$$d' "$$f"; \
	done
	@touch $@

DOC_PUBLIC+=doc/content/docs/reference/metrics/index.md
CLEANFILES+=doc/content/docs/reference/metrics
doc/content/docs/reference/metrics/index.md: hack/gen-metrics-doc.sh $(shell find pkg internal/pkg -name metrics.go) $(FRONT)
	@mkdir -p $(dir $@)
	$< > $@
	@$(FRONT) $@ 'title: Korrel8r Metrics' 'description: Metrics generated by korrel8r' 'weight: 50'

DOC_PUBLIC+=doc/content/docs/reference/mcp/index.md
CLEANFILES+=doc/content/docs/reference/mcp
doc/content/docs/reference/mcp/index.md: $(shell find pkg/mcp) cmd/korrel8r-docgen/main.go $(FRONT)
	@mkdir -p $(dir $@)
	go run ./cmd/korrel8r-docgen > $@
	@$(FRONT) $@ 'title: MCP API' 'description: MCP tool reference' 'weight: 90'

DOC_PUBLIC+=doc/content/docs/reference/template-functions.md
CLEANFILES+=doc/content/docs/reference/template-functions.md
doc/content/docs/reference/template-functions.md: pkg/engine/template_funcs.go $(MAKEFILE_LIST) $(FRONT) $(TOOL_DEPS)
	@mkdir -p $(dir $@)
	@echo > $@
	@$(GO_TOOL) gomarkdoc -u ./pkg/engine \
    -t 'package={{range .Types}}{{range .Methods}}{{if eq .Name "TemplateFuncs"}}{{template "doc" .Doc}}{{"\n\n"}}{{end}}{{end}}{{end}}' >> $@
	@$(GO_TOOL) gomarkdoc -u $(shell go list ./pkg/domains/...) \
     -t 'package={{range .Types}}{{range .Methods}}{{if eq .Name "TemplateFuncs"}}## {{$$.Name}}{{"\n"}}{{template "doc" .Doc}}{{"\n\n"}}{{end}}{{end}}{{end}}' >> $@
	@$(FRONT) $@ 'title: Template Functions' 'description: Functions available in rule templates' 'weight: 40'

doc/public: $(DOC_PUBLIC) $(shell find doc/* -name public -prune -o -print) $(TOOL_DEPS)
	$(GO_TOOL) hugo --quiet --source doc
	@touch $@
CLEANFILES+=doc/public
CLEANFILES+=$(DOC_PUBLIC)
