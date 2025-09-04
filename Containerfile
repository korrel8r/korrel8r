# Build the korrel8r binary
# Note: the Go version in this image must be compatible with ./go.mod
FROM registry.access.redhat.com/ubi9/go-toolset AS builder

USER 0
WORKDIR /src
# Download and cache go modules before building.
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy go sources and build
COPY cmd cmd
COPY pkg pkg
COPY internal internal

RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=1 GOOS=linux GOFLAGS="-mod=readonly -tags=strictfipsruntime,openssl" GOEXPERIMENT=strictfipsruntime go build -tags netgo ./cmd/korrel8r

# Commit build cache
RUN true

# Build a minimal runtime image
FROM registry.access.redhat.com/ubi9/ubi-minimal

WORKDIR /
COPY --from=builder /src/korrel8r /usr/bin/korrel8r
COPY etc/korrel8r /etc/korrel8r
USER 1000
ENTRYPOINT ["/usr/bin/korrel8r", "web"]
