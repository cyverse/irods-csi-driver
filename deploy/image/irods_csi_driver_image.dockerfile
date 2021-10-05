# iRODS-CSI-Driver-Image
#
# VERSION	1.0


##############################################
# irods-csi-driver image
##############################################
FROM ubuntu:18.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Driver Image"

ARG GOROOT=/usr/local/go
ARG CSI_DRIVER_SRC_DIR="/go/src/github.com/cyverse/irods-csi-driver"
ARG IRODS_FUSE_DIR="/opt/irodsfs"
ARG IRODS_FUSE_POOL_SERVER_DIR="/opt/irodsfs-pool"
ARG FUSE_NFS_DIR="/opt/fuse-nfs"
ARG DEBIAN_FRONTEND=noninteractive

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget fuse apt-transport-https lsb-release gnupg

# Setup NFS Client and WebDAV Client
RUN apt-get install -y nfs-common davfs2

WORKDIR /opt/

# Setup CSI Driver
COPY --from=irods_csi_driver_build:latest ${CSI_DRIVER_SRC_DIR}/bin/irods-csi-driver /bin/irods-csi-driver
# Setup iRODS FUSE Lite
COPY --from=irods_fuse_client_build:latest ${IRODS_FUSE_DIR}/mount_exec/mount.irodsfs /sbin/mount.irodsfs
COPY --from=irods_fuse_client_build:latest ${IRODS_FUSE_DIR}/bin/irodsfs /usr/bin/irodsfs
# Setup iRODS FUSE Lite Pool Server
COPY --from=irods_fuse_pool_server_build:latest ${IRODS_FUSE_POOL_SERVER_DIR}/bin/irodsfs-pool /usr/bin/irodsfs-pool

COPY start.sh /usr/bin/start.sh
RUN chmod 755 /usr/bin/start.sh

ENTRYPOINT ["/usr/bin/start.sh"]
