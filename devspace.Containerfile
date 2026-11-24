# Go-tools container for devspace to sync and auto-build korrel8r.
# Note: the Go version in this image must be compatible with ./go.mod
FROM registry.access.redhat.com/ubi10/go-toolset

USER 0
WORKDIR /src

# Pin cache paths so they match at build-time and runtime regardless of UID.
# OpenShift runs containers as random non-root UIDs; without explicit paths the
# runtime user gets different defaults and re-downloads everything.
ENV GOMODCACHE=/go/pkg/mod \
    GOCACHE=/go/cache/go-build \
    GONOSUMDB=* \
    GONOSUMCHECK=*

# Download and cache go modules before building.
COPY go.mod go.sum ./
COPY pkg/api/go.mod pkg/api/go.sum pkg/api/
COPY pkg/mcp/go.mod pkg/mcp/go.sum pkg/mcp/
RUN go mod download

# Copy go sources and pre-build to warm the build cache.
COPY cmd cmd
COPY pkg pkg
COPY internal internal
RUN go build ./...

# Ensure caches and work dirs are writable by any UID (OpenShift random UIDs).
RUN mkdir -p /.devspace && chmod -R a+rwX $GOMODCACHE $GOCACHE /src /.devspace
