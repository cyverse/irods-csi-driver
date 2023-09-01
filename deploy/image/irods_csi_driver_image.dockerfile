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
ARG FUSE_NFS_DIR="/opt/fuse-nfs"
ARG DEBIAN_FRONTEND=noninteractive
ARG IRODSFS_VER=v0.8.18

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget curl fuse apt-transport-https lsb-release gnupg

# Setup NFS Client and WebDAV Client
RUN apt-get install -y nfs-common davfs2

### Install irodsfs
RUN mkdir -p /tmp/irodsfs && \
    mkdir -p /var/lib/irodsfs
RUN curl -L https://github.com/cyverse/irodsfs/releases/download/${IRODSFS_VER}/irodsfs-${IRODSFS_VER}-linux-amd64.tar.gz --output /tmp/irodsfs/irodsfs.tar.gz
RUN tar zxvf /tmp/irodsfs/irodsfs.tar.gz -C /tmp/irodsfs && \
    cp /tmp/irodsfs/irodsfs /usr/bin && \
    rm -rf /tmp/irodsfs

ADD https://raw.githubusercontent.com/cyverse/irodsfs/${IRODSFS_VER}/mount_exec/mount.irodsfs \
  /sbin/mount.irodsfs
RUN chmod +x /sbin/mount.irodsfs

### Install dumb-init
ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
  /usr/bin/dumb-init
RUN chmod +x /usr/bin/dumb-init

WORKDIR /opt/

# Setup CSI Driver
COPY --from=irods_csi_driver_build:latest ${CSI_DRIVER_SRC_DIR}/bin/irods-csi-driver /usr/bin/irods-csi-driver

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/bin/irods-csi-driver"]