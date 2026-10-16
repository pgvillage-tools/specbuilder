# Build the manager binary
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0-devel

FROM docker.io/golang:bookworm AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -a \
    -ldflags="-X 'github.com/pgvillage-tools/specbuilder/cmd.Version=$VERSION'" -o specbuilder ./cmd/specbuilder

FROM docker.io/debian:bookworm-slim

LABEL MAINTAINER=Nibble-IT
WORKDIR /
COPY --from=builder /workspace/specbuilder ./

ENV ORIONAPI_CLUSTER_NAME=specbuilder \
    ORIONAPI_STORE_BACKEND=etcdv3

RUN useradd specbuilder
USER specbuilder

EXPOSE 8080

ENTRYPOINT ["/specbuilder"]
