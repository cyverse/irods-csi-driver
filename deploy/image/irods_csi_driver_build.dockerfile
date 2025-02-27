# iRODS-CSI-Driver-Build
#
# VERSION	1.0


##############################################
# Build irods-csi-driver
##############################################
FROM golang:1.23.6
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Driver Build Image"

ARG SRC_DIR="/go/src/github.com/cyverse/irods-csi-driver/"

WORKDIR ${SRC_DIR}

# Cache go modules
ENV GOPROXY=direct
COPY go.mod .
COPY go.sum .
RUN go mod download

ADD . .
RUN make irods-csi-driver

