
.PHONY: check
all: test lint

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run
