# Build the korrel8r binary
FROM golang:1.21.4 as builder

WORKDIR /workspace
# Download and cache go modules before building.
COPY go.mod go.sum .
RUN go mod download

# Copy go sources and build
COPY cmd cmd
COPY pkg pkg
COPY internal internal
RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-mod=readonly go build -tags netgo ./cmd/korrel8r
RUN true # Commit build cache

# TODO: using fedora image as a temporary workaround to install graphviz.
# Remove the graphviz dependency (separate web browser from REST API)
# When removed: Use gcr.io/distroless/static:nonroot as base.
# See https://github.com/GoogleContainerTools/distroless for more details
FROM quay.io/fedora/fedora
WORKDIR /
RUN dnf -y install graphviz
COPY --from=builder /workspace/korrel8r /bin/korrel8r
COPY etc/korrel8r/korrel8r.yaml /etc/korrel8r/korrel8r.yaml
COPY etc/korrel8r/rules /etc/korrel8r/rules
RUN useradd korrel8r
USER korrel8r
ENTRYPOINT ["/bin/korrel8r", "web"]
