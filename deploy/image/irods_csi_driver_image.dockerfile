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
ARG IRODS_FUSE_DIR="/opt/irodsfs"
ARG FUSE_NFS_DIR="/opt/fuse-nfs"
ARG DEBIAN_FRONTEND=noninteractive
ARG IRODSFS_VER=v0.12.3

### Install dumb-init
ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
    /usr/bin/dumb-init
RUN chmod +x /usr/bin/dumb-init

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget curl fuse apt-transport-https lsb-release gnupg

# Setup NFS Client and WebDAV Client
RUN apt-get install -y nfs-common davfs2

### Install fuse-overlayfs
ADD https://github.com/containers/fuse-overlayfs/releases/download/v1.13/fuse-overlayfs-x86_64 \
    /usr/bin/fuse-overlayfs
RUN chmod +x /usr/bin/fuse-overlayfs

ADD mount_exec/mount.fuseoverlayfs /sbin/mount.fuseoverlayfs
RUN chmod +x /sbin/mount.fuseoverlayfs

### Install irodsfs
RUN mkdir -p /tmp/irodsfs && \
    mkdir -p /var/lib/irodsfs
ADD https://github.com/cyverse/irodsfs/releases/download/${IRODSFS_VER}/irodsfs-${IRODSFS_VER}-linux-amd64.tar.gz \
    /tmp/irodsfs/irodsfs.tar.gz
RUN tar zxvf /tmp/irodsfs/irodsfs.tar.gz -C /tmp/irodsfs && \
    cp /tmp/irodsfs/irodsfs /usr/bin && \
    rm -rf /tmp/irodsfs

ADD mount_exec/mount.irodsfs /sbin/mount.irodsfs
RUN chmod +x /sbin/mount.irodsfs

WORKDIR /opt/

# Setup CSI Driver
COPY --from=irods_csi_driver_build:latest ${CSI_DRIVER_SRC_DIR}/bin/irods-csi-driver /usr/bin/irods-csi-driver

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/bin/irods-csi-driver"]