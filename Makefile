all: generate lint test build		## Generate code, lint, run all tests and build.

help: ## Describe make targets
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

build:				## Build the binary.
	go build -tags netgo ./cmd/korrel8r/.

lint:				## Check for lint.
	golangci-lint run

.PHONY: test

test:				## Run all the tests, requires a cluster.
	TEST_NO_SKIP=1 go test -cover -race ./...

cover:
	go test -coverprofile=test.cov  -race ./...
	go tool cover --html test.cov; sleep 2 # Sleep required to let browser start up.

generate:
	go generate -x ./...

web:
	go run ./cmd/korrel8r web -v3
