# Go-tools container for devspace to sync and auto-build korrel8r.
# See  target devspace in ./Makefile for more.

# Note: the Go version in this image must be compatible with ./go.mod
FROM registry.access.redhat.com/ubi9/go-toolset AS builder

USER 0
WORKDIR /src

# Put all the go caches under /src
ENV GOMODCACHE=/src/go-mod
ENV GOCACHE=/src/go-build
ENV GOBIN=/usr/bin
RUN mkdir -p $GOCACHE $GOMODCACHE

# Download and cache go modules.
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download -x

# Install gow: watches source changes and rebuilds go programs.
RUN go install github.com/mitranim/gow@latest

RUN mkdir -p /.devspace
RUN chmod -R g+rw /.devspace /src
