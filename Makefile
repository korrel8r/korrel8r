.PHONY: help
help: ## Describe make targets
	@grep '^[^: ]*: *.* *##' Makefile | sed 's/^\([^: ]*\): *.* *## \(.*\)$$/\1: \2/'

.PHONY: lint
lint:				## Check for lint.
	golangci-lint run

.PHONY: test
test:				## Run all the tests. Requires a cluster.
	TEST_NO_SKIP=1 go test -cover -race ./...
