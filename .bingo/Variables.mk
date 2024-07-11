# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for controller-gen variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(CONTROLLER_GEN)
#	@echo "Running controller-gen"
#	@$(CONTROLLER_GEN) <flags/args..>
#
CONTROLLER_GEN := $(GOBIN)/controller-gen-v0.14.0
$(CONTROLLER_GEN): $(BINGO_DIR)/controller-gen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/controller-gen-v0.14.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=controller-gen.mod -o=$(GOBIN)/controller-gen-v0.14.0 "sigs.k8s.io/controller-tools/cmd/controller-gen"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.59.1
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.59.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.59.1 "github.com/golangci/golangci-lint/cmd/golangci-lint"

KORREL8RCLI := $(GOBIN)/korrel8rcli-v0.0.2
$(KORREL8RCLI): $(BINGO_DIR)/korrel8rcli.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/korrel8rcli-v0.0.2"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=korrel8rcli.mod -o=$(GOBIN)/korrel8rcli-v0.0.2 "github.com/korrel8r/client/cmd/korrel8rcli"

KUSTOMIZE := $(GOBIN)/kustomize-v5.3.0
$(KUSTOMIZE): $(BINGO_DIR)/kustomize.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/kustomize-v5.3.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=kustomize.mod -o=$(GOBIN)/kustomize-v5.3.0 "sigs.k8s.io/kustomize/kustomize/v5"

SHFMT := $(GOBIN)/shfmt-v3.8.0
$(SHFMT): $(BINGO_DIR)/shfmt.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/shfmt-v3.8.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=shfmt.mod -o=$(GOBIN)/shfmt-v3.8.0 "mvdan.cc/sh/v3/cmd/shfmt"

SWAG := $(GOBIN)/swag-v1.16.3
$(SWAG): $(BINGO_DIR)/swag.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/swag-v1.16.3"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=swag.mod -o=$(GOBIN)/swag-v1.16.3 "github.com/swaggo/swag/cmd/swag"

SWAGGER := $(GOBIN)/swagger-v0.31.0
$(SWAGGER): $(BINGO_DIR)/swagger.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/swagger-v0.31.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=swagger.mod -o=$(GOBIN)/swagger-v0.31.0 "github.com/go-swagger/go-swagger/cmd/swagger"

