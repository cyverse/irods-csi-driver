# iRODS-CSI-Driver-Test-Image
#
# VERSION	1.0


##############################################
# irods-csi-driver-test image
##############################################
FROM ubuntu:22.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Driver Test Image"

ARG DEBIAN_FRONTEND=noninteractive

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y wget curl fuse apt-transport-https lsb-release gnupg

WORKDIR /opt/

# Create a test user and group
RUN groupadd -g 1000 testgroup && \
    useradd -u 1000 -g 1000 -ms /bin/bash testuser

# Set default command as bash
CMD ["/bin/bash"]
