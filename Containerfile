# Build the korrel8r binary
# Note: the Go version here should match the one in ./go.mod
FROM docker.io/golang:1.23 as builder
WORKDIR /src
# Download and cache go modules before building.
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy go sources and build
COPY cmd cmd
COPY pkg pkg
COPY internal internal
RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=0 GOOS=linux GOFLAGS=-mod=readonly go build -tags netgo ./cmd/korrel8r
RUN true # Commit build cache

FROM registry.access.redhat.com/ubi9/ubi-micro

WORKDIR /
COPY --from=builder /src/korrel8r /usr/bin/korrel8r
COPY etc/korrel8r /etc/korrel8r
USER 1000
ENTRYPOINT ["/usr/bin/korrel8r", "web"]
