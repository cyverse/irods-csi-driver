## iRODS CSI Driver Docker Images

The Dockerfiles in this directory are used to build the driver and package it for release.

- **irods_csi_driver_image.dockerfile** : iRODS CSI Driver Release (the host's irodsfsd performs filesystem-specific mounts)

The release Dockerfile builds the Go binary for Docker's target platform and
uses distribution packages only at runtime, so it supports both `linux/amd64`
and `linux/arm64`. Build a single-platform image with:

```shell
make image GOARCH=arm64
```

Publish a multi-architecture image manifest with:

```shell
make push-multiarch VERSION=v0.12.0
```

This requires a Docker Buildx builder with ARM emulation (QEMU) when the build
host is not ARM64, as the Ubuntu runtime image installs its packages during the
image build.
