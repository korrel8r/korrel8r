# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

## VERSION: Semantic version for release. Use a -dev suffix for work in progress.
VERSION?=0.6.2-dev1

## IMG: Base name of image to build or deploy, without version tag.
IMG?=quay.io/korrel8r/korrel8r
## IMGTOOL: May be podman or docker.
IMGTOOL?=$(or $(shell podman info > /dev/null 2>&1 && which podman), $(shell docker info > /dev/null 2>&1 && which docker))

# Setting GOENV
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

LOCALBIN ?= $(shell pwd)/tmp/bin

include .bingo/Variables.mk	# Versioned tools

check: generate lint test ## Lint and test code.

all: check install _site image-build ## Build and test everything locally. Recommended before pushing.

clean: ## Remove generated files, including checked-in files.
	rm -vrf bin _site $(GENERATED) $(shell find . -name 'zz_*')

VERSION_TXT=cmd/korrel8r/version.txt

ifneq ($(VERSION),$(file <$(VERSION_TXT)))
.PHONY: $(VERSION_TXT) # Force update if VERSION_TXT does not match $(VERSION)
endif
$(VERSION_TXT):
	echo $(VERSION) > $@

# List of generated files
GENERATED_DOC=doc/zz_domains.adoc doc/zz_rest_api.adoc doc/zz_api-ref.adoc
GENERATED=$(VERSION_TXT) pkg/config/zz_generated.deepcopy.go pkg/rest/zz_docs $(GENERATED_DOC) .copyright

generate: $(GENERATED) ## Generate code and doc.

GO_SRC=$(shell find . -name '*.go')

.copyright: $(GO_SRC)
	hack/copyright.sh	# Make sure files have copyright notice.
	@touch $@

pkg/config/zz_generated.deepcopy.go:  $(filter-out pkg/config/zz_generated.deepcopy.go,$(wildcard pkg/config/*.go)) $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object paths=./pkg/config/...

pkg/rest/zz_docs: $(wildcard pkg/rest/*.go pkg/korrel8r/*.go) $(SWAG)
	@mkdir -p $(dir $@)
	$(SWAG) init -q -g pkg/rest/api.go -o $@
	$(SWAG) fmt pkg/rest
	@touch $@


SHELLCHECK:= $(LOCALBIN)/shellcheck
$(SHELLCHECK):
	./hack/shellcheck.sh

lint: $(VERSION_TXT) $(GOLANGCI_LINT) $(SHFMT) $(SHELLCHECK) ## Run the linter to find and fix code style problems.
	$(GOLANGCI_LINT) run --fix
	$(SHFMT) -l -w ./**/*.sh
	go mod tidy
	$(SHELLCHECK) -x -S style hack/*.sh

install: $(VERSION_TXT) ## Build and install the korrel8r binary in $GOBIN.
	go install -tags netgo ./cmd/korrel8r

test: ## Run all tests, requires a cluster.
	$(MAKE) TEST_NO_SKIP=1 test-skip
test-skip: $(VERSION_TXT) ## Run all tests but skip those requiring a cluster if not logged in.
	go test -timeout=1m -race ./...

cover: ## Run tests and show code coverage in browser.
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

CONFIG=etc/korrel8r/korrel8r.yaml
run: $(GENERATED) ## Run `korrel8r web` using configuration in ./etc/korrel8r
	go run ./cmd/korrel8r web -c $(CONFIG) $(ARGS)

# Full name of image
IMAGE=$(IMG):$(VERSION)

image-build: $(VERSION_TXT) ## Build image locally, don't push.
	$(IMGTOOL) build --tag=$(IMAGE) -f Containerfile .

image: image-build ## Build and push image. IMG must be set to a writable image repository.
	$(IMGTOOL) push -q $(IMAGE)

image-name: ## Print the full image name and tag.
	@echo $(IMAGE)

WATCH=kubectl get events -A --watch-only& trap "kill %%" EXIT;

deploy: image $(KUSTOMIZE)	## Deploy to current cluster using kustomize.
	cd config; $(KUSTOMIZE) edit set image "quay.io/korrel8r/korrel8r=$(IMAGE)"
	kubectl apply -k config
	kubectl apply -k config/route || echo "skipping route"
	$(WATCH) kubectl wait -n korrel8r --for=condition=available --timeout=60s deployment.apps/korrel8r

undeploy:
	kubectl delete -k config

# Run asciidoctor from an image.
ADOC_RUN=$(IMGTOOL) run -iq -v./doc:/doc:z -v./_site:/_site:z quay.io/rhdevdocs/devspaces-documentation
# From github.com:darshandsoni/asciidoctor-skins.git
CSS?=adoc-readthedocs.css
ADOC_ARGS=-a revnumber=$(VERSION) -a allow-uri-read -a stylesdir=css -a stylesheet=$(CSS) -D/_site /doc/index.adoc

# _site is published to github pages by .github/workflows/asciidoctor-ghpages.yml.
_site: $(shell find doc) $(GENERATED_DOC) ## Generate the website HTML.
	@mkdir -p $@/etc
	@cp -r doc/images $@
	$(ADOC_RUN) asciidoctor $(ADOC_ARGS)
	$(ADOC_RUN) asciidoctor-pdf $(ADOC_ARGS) -o ebook.pdf
	$(and $(shell type -p linkchecker),linkchecker --check-extern --ignore-url 'https?://localhost[:/].*' _site)
	@touch $@

doc/zz_domains.adoc: $(shell find cmd/korrel8r-doc internal pkg -name '*.go')
	go run ./cmd/korrel8r-doc pkg/domains/* > $@

doc/zz_rest_api.adoc: pkg/rest/zz_docs $(shell find etc/swagger) $(SWAGGER)
	$(SWAGGER) -q generate markdown -T etc/swagger -f $</swagger.json --output $@

# TODO CRD API doc is incomplete, need to fix this up.
doc/zz_api-ref.adoc:
	curl -qf https://raw.githubusercontent.com/alanconway/operator/main/doc/zz_api-ref.adoc > doc/zz_api-ref.adoc

release: all image ## Create a release tag and latest image. Working tree must be clean.
	hack/tag-release.sh $(VERSION)
	$(IMGTOOL) push -q "$(IMAGE)" "$(IMG):latest"

Tools: $(BINGO) ## Download all tools needed for development
	$(BINGO) get
