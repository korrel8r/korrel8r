# Makefile is self-documenting, comments starting with '##' are extracted as help text.
help: ## Display this help.
	@echo; echo = Targets =
	@grep -E '^[A-Za-Z0-9_-]+:.*##' Makefile | sed 's/:.*##\s*/#/' | column -s'#' -t
	@echo; echo  = Variables =
	@grep -E '^## [A-Z0-9_]+: ' Makefile | sed 's/^## \([A-Z0-9_]*\): \(.*\)/\1#\2/' | column -s'#' -t

include ../.bingo/Variables.mk	# Versioned tools
include ../Variables.mk
IMG=$(OPERATOR_IMG)
IMAGE=$(OPERATOR_IMAGE)

# Bundle options.
DEFAULT_CHANNEL ?= stable
BUNDLE_CHANNELS ?= --channels=$(DEFAULT_CHANNEL)
BUNDLE_DEFAULT_CHANNEL ?= --default-channel=$(DEFAULT_CHANNEL)
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_IMAGE defines the image:tag used for the bundle.
BUNDLE_IMAGE ?= $(IMG)-bundle:$(VERSION)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

all: build test bundle  ## All local build & test.

images: all image-push bundle-push ## Build and push all images.

##@ Development

.PHONY: manifests
manifests: $(CONTROLLER_GEN) $(KUSTOMIZE) ## Generate ClusterRole and CustomResourceDefinition objects.
	cd config/default && $(KUSTOMIZE) edit set image controller=$(IMAGE)
	cd config/default && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	sed -i 's|value:.*|value: $(KORREL8R_IMAGE)|' config/default/manager_image_patch.yaml
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: $(CONTROLLER_GEN) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject methods.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

go.mod: $(find -name *.go)
	go mod tidy

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run the linter to find and fix code style problems.
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: manifests generate lint $(SETUP_ENVTEST) ## Run tests.
	$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) -p env); go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: go.mod manifests generate lint ## Build manager binary.
	go build -o bin/manager main.go

run: go.mod manifests generate lint install ## Build and run a controller from your host
	cd config/overlays/rbac_for_test && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	$(KUSTOMIZE) build config/overlays/rbac_for_test | kubectl apply -f -
	KORREL8R_IMAGE=$(KORREL8R_IMAGE) go run main.go

image-build:  ## Build the manager image.
	$(IMGTOOL) build -q  -t $(IMAGE) .
image-push: image-build ## Push the manager image.
	$(IMGTOOL) push -q $(IMAGE)

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	oc create ns $(NAMESPACE) || true
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found -f -

.PHONY: deploy
deploy: install image-push ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: $(KUSTOMIZE) ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found -f -

##@ Bundle

bundle: manifests $(OPERATOR_SDK) ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS) --extra-service-accounts korrel8r-instance
	$(OPERATOR_SDK) bundle validate ./bundle
	touch $@

WATCH=kubectl get events -n $(NAMESPACE) --watch-only& trap "kill %%" EXIT;

bundle-build: bundle ## Build the bundle image.
	$(IMGTOOL) build -q  -f bundle.Dockerfile -t $(BUNDLE_IMAGE) .
bundle-push: bundle-build ## Push the bundle image.
	$(IMGTOOL) push -q $(BUNDLE_IMAGE)
bundle-run: install bundle-push image-push $(OPERATOR_SDK)  ## Run the bundle image.
	$(OPERATOR_SDK) -n $(NAMESPACE) cleanup korrel8r || true
	oc create namespace $(NAMESPACE) || true
	$(WATCH) $(OPERATOR_SDK) -n $(NAMESPACE) run bundle $(BUNDLE_IMAGE)
bundle-cleanup: $(OPERATOR_SDK)
	$(OPERATOR_SDK) -n $(NAMESPACE) cleanup korrel8r || true

.PHONY: push-all
push-all: image-push bundle-push

push-latest: push-all
	docker push $(IMAGE) $(IMG):latest
	docker push $(BUNDLE_IMAGE) $(IMG)-bundle:latest

clean:
	rm -rf bundle bundle.Dockerfile $(GENERATED) $(shell find -name zz_generated.*)

clean-cluster: bundle-cleanup undeploy	## Remove all test artifacts from the cluster.
	oc delete ns/$(NAMESPACE) || true
	oc get -o name operator | grep korrel8r | xargs -r oc delete || true
	oc delete -l app.kubernetes.io/name=korrel8r clusterrolebinding,clusterrole || true



test-deploy: clean-cluster deploy ## Deploy via kustoize and run a smoke-test
	hack/smoketest.sh

test-bundle: clean-cluster bundle-run  ## Run the bundle and run a smoke-test
	hack/smoketest.sh

instance: install		## Create the korrel8r instance
	$(KUSTOMIZE) build config/samples | kubectl -n $(NAMESPACE) apply -f -
	$(WATCH) kubectl wait -n $(NAMESPACE) --for=condition=available --timeout=60s deployment.apps/korrel8r-instance

OPHUB?=$(error Set OPHUB to the path to your local community-operators-prod clone)
OPHUB_VERSION=$(OPHUB)/operators/korrel8r/$(VERSION)
operatorhub: bundle		## Generate modified bundle manifest for operator hub.
	mkdir -p $(OPHUB_VERSION)
	cp -aT bundle $(OPHUB_VERSION)
	echo -e '\n  # Annotations for OperatorHub\n  com.redhat.openshift.versions: "v4.10"' >> $(OPHUB_VERSION)/metadata/annotations.yaml
