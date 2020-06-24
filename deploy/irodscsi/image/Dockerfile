# iRODS-CSI-Image
#
# VERSION	1.0

FROM	ubuntu:18.04
LABEL maintainer="Illyoung Choi <iychoi@email.arizona.edu>"
LABEL version="0.1"
LABEL description="iRODS CSI Docker Image"

ARG GOLANG_VERSION=1.13.9
ARG GO_ARCH
ARG GOROOT=/usr/local/go
ARG SRC_DIR="/go/src/github.com/cyverse/irods-csi-driver/"
ARG DEBIAN_FRONTEND=noninteractive

##############################################
# Setup Utility Packages
##############################################
RUN apt-get update && \
    apt-get install -y sudo --option=Dpkg::Options::=--force-confdef && \
    apt-get install -y wget curl unzip python-pip build-essential fuse git apt-transport-https pkg-config libfuse-dev lsb-release

##############################################
# Setup iRODS Packages
##############################################
RUN wget -qO - https://packages.irods.org/irods-signing-key.asc | apt-key add -
RUN echo "deb [arch=amd64] https://packages.irods.org/apt/ $(lsb_release -sc) main" | tee /etc/apt/sources.list.d/renci-irods.list
RUN apt-get update && \
    apt-get install -y irods-dev irods-runtime \
                       irods-externals-cmake3.11.4-0 irods-externals-clang6.0-0 irods-externals-cppzmq4.2.3-0 \
                       irods-externals-libarchive3.3.2-1 irods-externals-avro1.9.0-0 irods-externals-boost1.67.0-0 \
                       irods-externals-clang-runtime6.0-0 irods-externals-jansson2.7-0 irods-externals-zeromq4-14.1.6-0 \
                       irods-icommands

##############################################
# Download cyverse-irods
##############################################
WORKDIR /opt/
RUN git clone https://github.com/cyverse/irods_client_fuse.git
WORKDIR /opt/irods_client_fuse

ENV LD_LIBRARY_PATH /opt/irods-externals/clang-runtime6.0-0/lib/
ENV PATH $PATH:/opt/irods-externals/cmake3.11.4-0/bin
RUN cmake -DCMAKE_INSTALL_PREFIX=/ . && \
    make && make install && \
    mkdir /root/.irods && \
    ln -s /usr/bin/irodsFs /sbin/mount.irodsfs

WORKDIR /opt/

##############################################
# Setup NFS Client and WebDAV Client
##############################################
RUN apt-get install -y nfs-common davfs2

##############################################
# Setup CSI Driver
##############################################
#RUN mkdir -p ${GOROOT} && \
#    curl https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-${GO_ARCH}.tar.gz | tar xzf - -C ${GOROOT} --strip-components=1

# Copy source directories
#COPY . ${SRC_DIR}
#RUN chmod -R

# Copy output
#COPY --from=builder ${SRC_DIR}/_output/irodscsi /usr/local/bin/irodscsi

#ENV GOROOT=${GOROOT} \
#    GOPATH=/go \
#    CGO_ENABLED=1 \
#    PATH="${GOROOT}/bin:${GOPATH}/bin:${PATH}"

#WORKDIR ${SRC_DIR}



# verify that all dynamically linked libraries are available
#RUN [ $(ldd /usr/local/bin/irodscsi | grep -c '=> not found') = '0' ]

#ENTRYPOINT ["/usr/local/bin/irodscsi"]
