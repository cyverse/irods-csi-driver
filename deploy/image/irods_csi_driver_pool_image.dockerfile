# iRODS-CSI-Driver-Pool-Image
#
# VERSION	1.0


##############################################
# irods-csi-driver-pool image
##############################################
FROM ubuntu:18.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Driver Pool Image"

ARG IRODS_FUSE_POOL_SERVER_DIR="/opt/irodsfs-pool"
ARG DEBIAN_FRONTEND=noninteractive

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget apt-transport-https lsb-release gnupg

### Install dumb-init
ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
  /usr/bin/dumb-init
RUN chmod +x /usr/bin/dumb-init

WORKDIR /opt/

# Setup iRODS FUSE Lite Pool Server
COPY --from=irods_csi_driver_pool_build:latest ${IRODS_FUSE_POOL_SERVER_DIR}/bin/irodsfs-pool /usr/bin/irodsfs-pool

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/bin/irodsfs-pool", "-f"]