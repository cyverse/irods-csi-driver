# iRODS-FUSE-Client-Build
#
# VERSION	1.0


##############################################
# Build irods-fuse
##############################################
FROM ubuntu:18.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS FUSE Client Build Image"

ARG DEBIAN_FRONTEND=noninteractive

# Setup Utility Packages
RUN apt-get update && \
    apt-get install -y sudo --option=Dpkg::Options::=--force-confdef && \
    apt-get install -y wget build-essential fuse git apt-transport-https pkg-config libfuse-dev lsb-release

# Setup iRODS Packages
RUN wget -qO - https://packages.irods.org/irods-signing-key.asc | apt-key add -
RUN echo "deb [arch=amd64] https://packages.irods.org/apt/ $(lsb_release -sc) main" | tee /etc/apt/sources.list.d/renci-irods.list
RUN apt-get update && \
    apt-get install -y irods-dev irods-runtime \
                       irods-externals-cmake3.11.4-0 irods-externals-clang6.0-0 irods-externals-cppzmq4.2.3-0 \
                       irods-externals-libarchive3.3.2-1 irods-externals-avro1.9.0-0 irods-externals-boost1.67.0-0 \
                       irods-externals-clang-runtime6.0-0 irods-externals-jansson2.7-0 irods-externals-zeromq4-14.1.6-0

# Download cyverse-irods
WORKDIR /opt/
RUN git clone https://github.com/cyverse/irods_client_fuse.git
WORKDIR /opt/irods_client_fuse

ENV LD_LIBRARY_PATH /opt/irods-externals/clang-runtime6.0-0/lib/
ENV PATH $PATH:/opt/irods-externals/cmake3.11.4-0/bin

# Build
RUN cmake -DCMAKE_INSTALL_PREFIX=/ . && \
    make && make install

