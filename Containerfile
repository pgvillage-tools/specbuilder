ARG VERSION=v0.0.0-devel

# Build on the native platform of the builder and cross-compile for the target platform.
# TARGETOS and TARGETARCH are set by buildx (from --platform), and must not have a default here,
# or that default overrides the value from buildx and every image ends up with the same binary.
FROM --platform=$BUILDPLATFORM docker.io/golang:bookworm AS builder

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
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -v -a \
    -ldflags="-X 'github.com/pgvillage-tools/specbuilder/cmd.Version=$VERSION'" -o specbuilder ./cmd/specbuilder

FROM docker.io/debian:bookworm-slim

LABEL MAINTAINER=Nibble-IT

# specbuilder calls the GitHub API over https
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /
COPY --from=builder /workspace/specbuilder ./

RUN useradd specbuilder
USER specbuilder

ENTRYPOINT ["/specbuilder"]
