#! /bin/bash

echo "Starting iRODS FUSE Lite Pool Service"
# execute irodsfs-pool
/usr/bin/irodsfs-pool

echo "Starting iRODS CSI Driver"
# execute irods-csi-driver
/bin/irods-csi-driver "$@"