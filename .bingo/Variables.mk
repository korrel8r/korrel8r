# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for bingo variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BINGO)
#	@echo "Running bingo"
#	@$(BINGO) <flags/args..>
#
BINGO := $(GOBIN)/bingo-v0.9.0
$(BINGO): $(BINGO_DIR)/bingo.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/bingo-v0.9.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=bingo.mod -o=$(GOBIN)/bingo-v0.9.0 "github.com/bwplotka/bingo"

CONTROLLER_GEN := $(GOBIN)/controller-gen-v0.14.0
$(CONTROLLER_GEN): $(BINGO_DIR)/controller-gen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/controller-gen-v0.14.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=controller-gen.mod -o=$(GOBIN)/controller-gen-v0.14.0 "sigs.k8s.io/controller-tools/cmd/controller-gen"

CRD_REF_DOCS := $(GOBIN)/crd-ref-docs-v0.0.11
$(CRD_REF_DOCS): $(BINGO_DIR)/crd-ref-docs.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/crd-ref-docs-v0.0.11"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=crd-ref-docs.mod -o=$(GOBIN)/crd-ref-docs-v0.0.11 "github.com/elastic/crd-ref-docs"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.57.1
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.57.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.57.1 "github.com/golangci/golangci-lint/cmd/golangci-lint"

KUSTOMIZE := $(GOBIN)/kustomize-v5.3.0
$(KUSTOMIZE): $(BINGO_DIR)/kustomize.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/kustomize-v5.3.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=kustomize.mod -o=$(GOBIN)/kustomize-v5.3.0 "sigs.k8s.io/kustomize/kustomize/v5"

OPERATOR_SDK := $(GOBIN)/operator-sdk-v1.34.1
$(OPERATOR_SDK): $(BINGO_DIR)/operator-sdk.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/operator-sdk-v1.34.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=operator-sdk.mod -o=$(GOBIN)/operator-sdk-v1.34.1 "github.com/operator-framework/operator-sdk/cmd/operator-sdk"

SETUP_ENVTEST := $(GOBIN)/setup-envtest-v0.0.0-20240320141353-395cfc7486e6
$(SETUP_ENVTEST): $(BINGO_DIR)/setup-envtest.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/setup-envtest-v0.0.0-20240320141353-395cfc7486e6"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=setup-envtest.mod -o=$(GOBIN)/setup-envtest-v0.0.0-20240320141353-395cfc7486e6 "sigs.k8s.io/controller-runtime/tools/setup-envtest"

SWAG := $(GOBIN)/swag-v1.16.3
$(SWAG): $(BINGO_DIR)/swag.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/swag-v1.16.3"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=swag.mod -o=$(GOBIN)/swag-v1.16.3 "github.com/swaggo/swag/cmd/swag"

SWAGGER := $(GOBIN)/swagger-v0.30.5
$(SWAGGER): $(BINGO_DIR)/swagger.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/swagger-v0.30.5"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=swagger.mod -o=$(GOBIN)/swagger-v0.30.5 "github.com/go-swagger/go-swagger/cmd/swagger"

