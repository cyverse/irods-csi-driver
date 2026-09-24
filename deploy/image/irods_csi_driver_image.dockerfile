# iRODS-CSI-Driver-Image
#
# VERSION	1.0


##############################################
# Build irods-csi-driver for the requested target platform
##############################################
FROM --platform=$BUILDPLATFORM golang:1.26.8 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=v0.12.1
ARG SRC_DIR="/go/src/github.com/cyverse/irods-csi-driver/"

WORKDIR ${SRC_DIR}
ENV GOPROXY=direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-$(go env GOARCH)} make irods-csi-driver VERSION=${VERSION}

##############################################
# irods-csi-driver image
##############################################
FROM ubuntu:22.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Driver Image"

ARG DEBIAN_FRONTEND=noninteractive

# mount(8) is required for CSI bind mounts. Filesystem-specific mounts are
# delegated to the host's irodsfsd service.
RUN apt-get update && \
    apt-get install -y --no-install-recommends dumb-init util-linux && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /opt/

# Setup CSI Driver
COPY --from=builder /go/src/github.com/cyverse/irods-csi-driver/bin/irods-csi-driver /usr/bin/irods-csi-driver

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/bin/irods-csi-driver"]
