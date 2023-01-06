.PHONY: help
help: ## Describe make targets
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

all: lint test			## Run all tests.

.PHONY: lint
lint:				## Check for lint.
	golangci-lint run

.PHONY: test
test:				## Run all the tests, don't use test cache. Requires a cluster.
	TEST_NO_SKIP=1 go test -count=1 -cover -race ./...
