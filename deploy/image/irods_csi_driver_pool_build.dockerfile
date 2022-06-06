# iRODSFS-Pool-Server-Build
#
# VERSION	1.0


##############################################
# Build irodsfs-pool
##############################################
FROM golang:1.16.8-stretch
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS FUSE Lite Pool Server Build Image"

# Download github.com/cyverse/irodsfs-pool
WORKDIR /opt/
RUN git clone https://github.com/cyverse/irodsfs-pool.git
WORKDIR /opt/irodsfs-pool
RUN git checkout tags/v0.5.1

# Build
RUN make build