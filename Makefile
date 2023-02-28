
help: ## Describe make targets
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

all: generate lint test build		## Generate code, lint, run all tests and build.

build:				## Build the binary.
	go build -tags netgo ./cmd/korrel8r/.

lint:				## Check for lint.
	golangci-lint run

.PHONY: test
test:				## Run all the tests, no cache, requires a cluster.
	TEST_NO_SKIP=1 go test -count=1 -cover -race ./...

generate:
	go generate -x ./...
