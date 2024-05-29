# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release, use -dev for development pre-release versions.
VERSION?=0.6.4
## IMG_ORG: org name for images, for example quay.io/alanconway.
IMG_ORG?=$(error Set IMG_ORG to organization prefix for images, e.g. IMG_ORG=quay.io/alanconway)
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell podman info > /dev/null 2>&1 && which podman), $(shell docker info > /dev/null 2>&1 && which docker))
## NAMESPACE: Namespace for `make deploy`
NAMESPACE=korrel8r
## CONFIG: Configuration file for `make run`
CONFIG?=etc/korrel8r/openshift-route.yaml

# Name of image.
IMG?=$(IMG_ORG)/korrel8r
IMAGE=$(IMG):$(VERSION)

# Setting GOENV
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

LOCALBIN ?= $(shell pwd)/tmp/bin
$(shell mkdir -p $(LOCALBIN))
PATH:=$(LOCALBIN):$(PATH)

include .bingo/Variables.mk	# Versioned tools

check: lint test ## Lint and test code.

all: check install _site image-build ## Build and test everything locally. Recommended before pushing.

clean: ## Remove generated files, including checked-in files.
	rm -rf bin _site $(GENERATED) doc/gen tmp

# Generated files
VERSION_TXT=internal/pkg/build/version.txt
SWAGGER_SPEC=pkg/rest/docs/swagger.json
SWAGGER_CLIENT=client/pkg/swagger

GEN_SRC=$(VERSION_TXT) $(SWAGGER_SPEC) $(SWAGGER_CLIENT) pkg/config/zz_generated.deepcopy.go .copyright

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

generate: $(GEN_SRC) ## Generate code and doc.

.copyright: $(shell find . -name '*.go')
	hack/copyright.sh	# Make sure files have copyright notice.
	@touch $@

pkg/config/zz_generated.deepcopy.go:  $(filter-out pkg/config/zz_generated.deepcopy.go,$(wildcard pkg/config/*.go)) $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object paths=./pkg/config/...

$(SWAGGER_SPEC): $(wildcard pkg/rest/*.go) $(SWAG)
	@mkdir -p $(dir $@)
	$(SWAG) init -q -g pkg/rest/operations.go -o $(dir $@)
	$(SWAG) fmt pkg/rest
	@touch $@

SHELLCHECK:= $(LOCALBIN)/shellcheck
$(SHELLCHECK):
	./hack/shellcheck.sh

lint: generate $(GOLANGCI_LINT) $(SHFMT) $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	$(GOLANGCI_LINT) run --fix
	$(SHFMT) -l -w ./**/*.sh
	go mod tidy
	$(SHELLCHECK) -x -S style hack/*.sh

install: $(KORREL8R) $(KORREL8RCLI) ## Build and install binaries in $GOBIN.

KORREL8R=$(GOBIN)/korrel8r
$(KORREL8R): $(GEN_SRC)
	go install -tags netgo ./cmd/korrel8r

KORREL8RCLI=$(GOBIN)/korrel8rcli
$(KORREL8RCLI): $(GEN_SRC)
	go install -tags netgo ./client/cmd/korrel8rcli

.PHONY: test
test: ## Run all tests, requires a cluster.
	$(MAKE) TEST_NO_SKIP=1 test-skip

test-skip: $(GEN_SRC) ## Run all tests but skip those requiring a cluster if not logged in.
	go test -timeout=1m -race ./...

cover: ## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

run: $(GEN_SRC) ## Run `korrel8r web` using configuration in ./etc/korrel8r
	go run ./cmd/korrel8r web -c $(CONFIG) $(ARGS)

image-build: $(GEN_SRC) ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) push -q $(IMAGE)

image-name: ## Print the full image name and tag.
	@echo $(IMAGE)

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

deploy-ns: image kustomize-edit	## Deploy only namespace-scoped resources.
	@rm -rf tmp/resources; mkdir -p tmp/resources
	kustomize build config -o tmp/resources
	kubectl apply -f tmp/resources/apps_v1_deployment_korrel8r.yaml -f tmp/resources/v1_service_korrel8r.yaml
	$(DEPLOY_ROUTE)
	$(WAIT_DEPLOYMENT)

restart:			## Force restart of pods.
	kubectl get -n $(NAMESPACE) pod -o name | xargs -r oc delete -n $(NAMESPACE)
	$(WAIT_DEPLOYMENT)

undeploy:
	@kubectl delete -k config/route || true
	@kubectl delete -k config || true




ASCIIDOCTOR:=$(LOCALBIN)/asciidoctor
$(ASCIIDOCTOR):
	gem install asciidoctor --user-install --bindir $(LOCALBIN)

# From github.com:darshandsoni/asciidoctor-skins.git
CSS?=adoc-readthedocs.css
ADOC_FLAGS=-a allow-uri-read -a stylesdir=$(shell pwd)/doc/css -a stylesheet=$(CSS)  -a revnumber=$(VERSION) -a revdate=$(shell date -I)

# _site is published to github pages by .github/workflows/asciidoctor-ghpages.yml.
_site: doc _site/man $(ASCIIDOCTOR) ## Generate the website HTML.
	@mkdir -p $@/etc
	@cp -r doc/images $@
	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D_site doc/index.adoc


	$(ASCIIDOCTOR) $(ADOC_FLAGS) -D_site/gen/cmd doc/gen/cmd/*.adoc
	$(and $(shell type -p linkchecker),linkchecker --check-extern --ignore-url 'https?://localhost[:/].*' _site)
	@touch $@

doc: $(shell find doc -type f) doc/gen/domains.adoc doc/gen/rest_api.adoc doc/gen/cmd
	touch $@

doc/gen/domains.adoc: $(shell find cmd/korrel8r-doc internal pkg -name '*.go') $(GEN_SRC)
	@mkdir -p $(dir $@)
	go run ./cmd/korrel8r-doc pkg/domains/* > $@

doc/gen/rest_api.adoc: $(SWAGGER_SPEC) $(shell find etc/swagger) $(SWAGGER)
	@mkdir -p $(dir $@)
	$(SWAGGER) -q generate markdown -T etc/swagger -f $(SWAGGER_SPEC) --output $@

_site/man: $(GEN_SRC)	## Generated man pages documentation.
	@mkdir -p $@
	go run ./cmd/korrel8r doc man $@
	go run ./client/cmd/korrel8rcli doc man $@
	touch $@

KRAMDOC:=$(LOCALBIN)/kramdoc
$(KRAMDOC):
	gem install kramdown-asciidoc --user-install --bindir $(LOCALBIN)

doc/gen/cmd: $(GEN_SRC) $(KRAMDOC) ## Generated command documentation
	@mkdir -p $@
	go run ./cmd/korrel8r doc markdown $@
	go run ./client/cmd/korrel8rcli doc markdown $@
	cd $@ && for F in $$(basename -s .md *.md); do $(KRAMDOC) --heading-offset=-1 -o $$F.adoc $$F.md; done
	rm $@/*.md
	touch $@

pre-release: all image ## Set VERISON and IMG_ORG to build release artifacts. Commit before doing "make release".

release: pre-release		## Set VERISON and IMG_ORG to push release tags and images.
	hack/tag-release.sh $(VERSION)
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"

tools: $(BINGO) $(ASCIIDOCTOR) $(KRAMDOC) ## Download all tools needed for development
	$(BINGO) get
	go mod tidy

# The REST client package is in a separate Go module with minimal dependencies.

$(SWAGGER_CLIENT): $(SWAGGER_SPEC) $(SWAGGER) ## Generate client packages.
	mkdir -p $@
	cd $@ && $(SWAGGER) generate -q client -f $(abspath $(SWAGGER_SPEC)) && go mod tidy
	touch $@
