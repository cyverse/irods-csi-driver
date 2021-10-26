## iRODS CSI Driver Docker Images

Dockerfiles in this directory is used to build the driver or packaging the driver for release.

- **irods_fuse_build.dockerfile** : Build iRODS FUSE Client
- **irods_fuse_pool_server_build.dockerfile** : Build iRODS FUSE Pool Server
- **irods_csi_driver_pool_build.dockerfile** : Build iRODS FUSE Pool Server for CSI Driver
- **irods_csi_driver_build.dockerfile** : Build iRODS CSI Driver
- **irods_csi_driver_pool_image.dockerfile** : iRODS FUSE Pool Server Release package (includes iRODS FUSE Pool Server)
- **irods_csi_driver_image.dockerfile** : iRODS CSI Driver Release package (includes CSI Driver code, iRODS-FUSE, NFS Client and Webdav Client (Davfs2))
