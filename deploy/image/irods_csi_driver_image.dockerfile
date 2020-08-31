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
ARG CSI_DRIVER_SRC_DIR="/go/src/github.com/cyverse/irods-csi-driver/"
ARG IRODS_FUSE_DIR="/opt/irods_client_fuse/"
ARG FUSE_NFS_DIR="/opt/fuse-nfs/"
ARG DEBIAN_FRONTEND=noninteractive

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget fuse apt-transport-https lsb-release gnupg python3-pip

# Setup iRODS Packages
RUN wget -qO - https://packages.irods.org/irods-signing-key.asc | apt-key add -
RUN echo "deb [arch=amd64] https://packages.irods.org/apt/ $(lsb_release -sc) main" | tee /etc/apt/sources.list.d/renci-irods.list
RUN apt-get update && \
    apt-get install -y irods-icommands

# Setup NFS Client and WebDAV Client
RUN apt-get install -y nfs-common davfs2

ENV LD_LIBRARY_PATH /opt/irods-externals/clang-runtime6.0-0/lib/

RUN mkdir /root/.irods && \
    echo "LD_LIBRARY_PATH=/opt/irods-externals/clang-runtime6.0-0/lib/" > /etc/irodsfs_env && \
    echo "HOME=/root" >> /etc/irodsfs_env

RUN pip3 install -q python-irodsclient

WORKDIR /opt/

# Setup CSI Driver
COPY --from=irods_csi_driver_build:latest ${CSI_DRIVER_SRC_DIR}/bin/irods-csi-driver /bin/irods-csi-driver
# Setup iRODS FUSE
COPY --from=irods_fuse_client_build:latest ${IRODS_FUSE_DIR}/bin/mount.irodsfs /sbin/mount.irodsfs
COPY --from=irods_fuse_client_build:latest ${IRODS_FUSE_DIR}/irodsFs /usr/bin/irodsFs
# Setup iRODS Python Scripts
COPY exec/irods_ls.py /usr/local/bin/irods_ls.py
COPY exec/irods_mkdir.py /usr/local/bin/irods_mkdir.py
COPY exec/irods_rmdir.py /usr/local/bin/irods_rmdir.py

ENTRYPOINT ["/bin/irods-csi-driver"]
