# Go-tools container for devspace to sync and auto-build korrel8r.
# Note: the Go version in this image must be compatible with ./go.mod
FROM registry.access.redhat.com/ubi10/go-toolset AS builder

USER 0
WORKDIR /src

# Download and cache go modules before building.
COPY go.mod go.sum ./
COPY pkg/api/go.mod pkg/api/go.sum pkg/api/
COPY pkg/mcp/go.mod pkg/mcp/go.sum pkg/mcp/
RUN go mod download -x

# Copy go sources and build
COPY cmd cmd
COPY pkg pkg
COPY internal internal

RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=1 GOOS=linux GOFLAGS="-mod=readonly -tags=strictfipsruntime,openssl" GOEXPERIMENT=strictfipsruntime go build -tags netgo ./cmd/korrel8r

# Commit build cache
RUN true

# Create devspace workdir
RUN mkdir -p /.devspace
RUN chmod -R g+rw /.devspace /src
