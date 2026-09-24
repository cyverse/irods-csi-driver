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
make release VERSION=v0.12.1
```

This requires a Docker Buildx builder with ARM emulation (QEMU) when the build
host is not ARM64, as the Ubuntu runtime image installs its packages during the
image build.

Creating a GitHub Release automatically packages the Helm chart and attaches
the `.tgz` file to that release. It also builds the AMD64 and ARM64 driver
images. To publish those images to Docker Hub, configure the repository secrets
`DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`; without them, the workflow performs
the multi-architecture build as a verification step only. The release tag must
match `v` followed by the Helm chart version (for example, `v0.12.1`).
