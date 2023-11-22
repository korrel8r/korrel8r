# Make rules for downloading tools needed for development
define TOOL_PKGS
github.com/go-swagger/go-swagger/cmd/swagger
github.com/swaggo/swag/cmd/swag
github.com/golangci/golangci-lint/cmd/golangci-lint
sigs.k8s.io/kind
sigs.k8s.io/controller-tools/cmd/controller-gen
endef

define TOOL_TARGET
bin/$(notdir $(1)):
	mkdir -p bin && GOBIN=$(abspath bin) go install $(1)@latest
endef

$(foreach pkg,$(TOOL_PKGS),$(eval $(call TOOL_TARGET,$(pkg))))

tools:  $(patsubst %,bin/%,$(notdir $(TOOL_PKGS)))
