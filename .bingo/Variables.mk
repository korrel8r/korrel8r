# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.9. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for benchstat variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(BENCHSTAT)
#	@echo "Running benchstat"
#	@$(BENCHSTAT) <flags/args..>
#
BENCHSTAT := $(GOBIN)/benchstat-v0.0.0-20250106172127-400946f43c82
$(BENCHSTAT): $(BINGO_DIR)/benchstat.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/benchstat-v0.0.0-20250106172127-400946f43c82"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=benchstat.mod -o=$(GOBIN)/benchstat-v0.0.0-20250106172127-400946f43c82 "golang.org/x/perf/cmd/benchstat"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v2.7.1
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v2.7.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v2.7.1 "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"

KORREL8RCLI := $(GOBIN)/korrel8rcli-v0.0.3
$(KORREL8RCLI): $(BINGO_DIR)/korrel8rcli.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/korrel8rcli-v0.0.3"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=korrel8rcli.mod -o=$(GOBIN)/korrel8rcli-v0.0.3 "github.com/korrel8r/client/cmd/korrel8rcli"

KUSTOMIZE := $(GOBIN)/kustomize-v5.5.0
$(KUSTOMIZE): $(BINGO_DIR)/kustomize.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/kustomize-v5.5.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=kustomize.mod -o=$(GOBIN)/kustomize-v5.5.0 "sigs.k8s.io/kustomize/kustomize/v5"

OAPI_CODEGEN := $(GOBIN)/oapi-codegen-v2.5.1
$(OAPI_CODEGEN): $(BINGO_DIR)/oapi-codegen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/oapi-codegen-v2.5.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=oapi-codegen.mod -o=$(GOBIN)/oapi-codegen-v2.5.1 "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"

SHFMT := $(GOBIN)/shfmt-v3.10.0
$(SHFMT): $(BINGO_DIR)/shfmt.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/shfmt-v3.10.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=shfmt.mod -o=$(GOBIN)/shfmt-v3.10.0 "mvdan.cc/sh/v3/cmd/shfmt"

