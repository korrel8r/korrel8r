# Go-tools container for devspace to sync and auto-build korrel8r.
# See  target devspace in ./Makefile for more.

# Note: the Go version in this image must be compatible with ./go.mod
# devspace.Containerfile should be updated to the same image.
FROM registry.access.redhat.com/ubi10/go-toolset AS builder

USER 0
WORKDIR /src

# Put all the go caches under /src
ENV GOMODCACHE=/src/go-mod
ENV GOCACHE=/src/go-build
ENV GOBIN=/usr/bin
RUN mkdir -p $GOCACHE $GOMODCACHE

# Download and cache go modules before building.
COPY go.mod go.sum ./
COPY pkg/api/go.mod pkg/api/go.sum pkg/api/
COPY pkg/mcp/go.mod pkg/mcp/go.sum pkg/mcp/
RUN go mod download -x

RUN mkdir -p /.devspace
RUN chmod -R g+rw /.devspace /src
