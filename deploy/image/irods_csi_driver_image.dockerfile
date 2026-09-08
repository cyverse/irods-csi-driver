# iRODS-CSI-Driver-Image
#
# VERSION	1.0


##############################################
# irods-csi-driver image
##############################################
FROM ubuntu:22.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Driver Image"

ARG GOROOT=/usr/local/go
ARG CSI_DRIVER_SRC_DIR="/go/src/github.com/cyverse/irods-csi-driver"
ARG DEBIAN_FRONTEND=noninteractive

### Install dumb-init
ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
    /usr/bin/dumb-init
RUN chmod +x /usr/bin/dumb-init

# mount(8) is required for CSI bind mounts. Filesystem-specific mounts are
# delegated to the host's irodsfsd service.
RUN apt-get update && \
    apt-get install -y --no-install-recommends util-linux && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /opt/

# Setup CSI Driver
COPY --from=irods_csi_driver_build:latest ${CSI_DRIVER_SRC_DIR}/bin/irods-csi-driver /usr/bin/irods-csi-driver

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/bin/irods-csi-driver"]
