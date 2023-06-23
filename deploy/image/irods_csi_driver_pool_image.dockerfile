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

ARG DEBIAN_FRONTEND=noninteractive
ARG IRODSFS_POOL_VER=v0.6.13

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget curl apt-transport-https lsb-release gnupg

### Install irodsfs-pool
RUN mkdir -p /tmp/irodsfs-pool && \
    mkdir -p /var/lib/irodsfs-pool
RUN curl -L https://github.com/cyverse/irodsfs-pool/releases/download/${IRODSFS_POOL_VER}/irodsfs-pool-${IRODSFS_POOL_VER}-linux-amd64.tar.gz --output /tmp/irodsfs-pool/irodsfs-pool.tar.gz
RUN tar zxvf /tmp/irodsfs-pool/irodsfs-pool.tar.gz -C /tmp/irodsfs-pool && \
    cp /tmp/irodsfs-pool/irodsfs-pool /usr/bin && \
    rm -rf /tmp/irodsfs-pool

### Install dumb-init
ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.5/dumb-init_1.2.5_x86_64 \
  /usr/bin/dumb-init
RUN chmod +x /usr/bin/dumb-init

WORKDIR /opt/

ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/bin/irodsfs-pool", "-f", "--data_root", "/var/lib/irodsfs-pool"]