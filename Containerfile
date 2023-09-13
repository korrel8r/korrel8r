# Build the korrel8r binary
FROM golang:1.20.1 as builder

WORKDIR /workspace
# Download and cache go modules before building.
COPY go.mod go.sum .
RUN go mod download

# Copy go sources and build
COPY cmd cmd
COPY pkg pkg
COPY internal internal
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly ./cmd/korrel8r

# TODO: Use distroless as minimal base image to package the binary.
# See https://github.com/GoogleContainerTools/distroless for more details
# gcr.io/distroless/static:nonroot
FROM quay.io/fedora/fedora
WORKDIR /
RUN dnf -y install graphviz
COPY --from=builder /workspace/korrel8r /bin/korrel8r
COPY etc/korrel8r/korrel8r.yaml /etc/korrel8r/korrel8r.yaml
COPY etc/korrel8r/rules /etc/korrel8r/rules
RUN useradd korrel8r
USER korrel8r
ENTRYPOINT ["/bin/korrel8r", "web"]
