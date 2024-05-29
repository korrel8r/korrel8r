# Build the korrel8r binary
FROM docker.io/golang:1.21.9 as builder
WORKDIR /src
# Download and cache go modules before building.
COPY go.mod go.mod
COPY go.sum go.sum
COPY client client
RUN go mod download

# Copy go sources and build
COPY cmd cmd
COPY pkg pkg
COPY internal internal
RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=0 GOOS=linux GOFLAGS=-mod=readonly go build -tags netgo ./cmd/korrel8r
RUN true # Commit build cache

# See https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi9/ubi-micro
WORKDIR /
COPY --from=builder /src/korrel8r /bin/korrel8r
COPY etc/korrel8r /etc/korrel8r
USER 1000
ENTRYPOINT ["/bin/korrel8r", "web"]
