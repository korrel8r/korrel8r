
help: ## Describe make targets
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

all: generate lint test		## Generate code, lint, run all tests.

lint:				## Check for lint.
	golangci-lint run

.PHONY: test
test:				## Run all the tests, no cache, requires a cluster.
	TEST_NO_SKIP=1 go test -count=1 -cover -race ./...

generate:
	go generate -x ./...
