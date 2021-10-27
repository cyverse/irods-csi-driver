# iRODSFS-Build
#
# VERSION	1.0


##############################################
# Build irodsfs
##############################################
FROM golang:1.16.8-stretch
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS FUSE Lite Build Image"

# Download github.com/cyverse/irodsfs
WORKDIR /opt/
RUN git clone https://github.com/cyverse/irodsfs.git
WORKDIR /opt/irodsfs
RUN git checkout tags/v0.3.16

# Build
RUN make build