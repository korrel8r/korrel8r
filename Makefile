all: generate lint test build		## Generate code, lint, run all tests and build.

help: ## Describe make targets
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

build:				## Build the binary.
	go build -tags netgo ./cmd/korrel8r/.

lint:				## Check for lint.
	golangci-lint run --fix

.PHONY: test

test:				## Run all the tests, requires a cluster.
	TEST_NO_SKIP=1 go test -timeout=1m -race ./...

cover:
	go test -coverprofile=test.cov ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

generate:
	go generate -x ./...
	hack/copyright.sh

tag:	     ## Create a version tag on the current branch, set TAG=vX.Y.Z.
	@echo "tagging $(or $(TAG),$(error Set version tag like: TAG=vX.Y.Z))"
	@echo -e 'package main\nfunc Version() string { return "$(TAG)"; }' > cmd/korrel8r/version.go
	go mod tidy
	$(MAKE)
	git add go.mod go.sum cmd/korrel8r/version.go
	git commit -m "changes for $(TAG)"
	git tag $(TAG)  -m "version $(TAG)"

tools:
	go install \
		github.com/go-swagger/go-swagger/cmd/swagger@latest \
		github.com/swaggo/swag/cmd/swag@latest
