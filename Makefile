PKG=github.com/cyverse/irods-csi-driver
CSI_DRIVER_BUILD_IMAGE=irods_csi_driver_build
CSI_DRIVER_BUILD_DOCKERFILE=deploy/image/irods_csi_driver_build.dockerfile
CSI_DRIVER_IMAGE?=cyverse/irods-csi-driver
CSI_DRIVER_DOCKERFILE=deploy/image/irods_csi_driver_image.dockerfile
CSI_DRIVER_POOL_IMAGE?=cyverse/irods-csi-driver-pool
CSI_DRIVER_POOL_DOCKERFILE=deploy/image/irods_csi_driver_pool_image.dockerfile
VERSION=v0.9.7
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X ${PKG}/pkg/common.driverVersion=${VERSION} -X ${PKG}/pkg/common.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/common.buildDate=${BUILD_DATE}"
GO111MODULE=on
GOPROXY=direct
GOPATH=$(shell go env GOPATH)

.EXPORT_ALL_VARIABLES:

.PHONY: irods-csi-driver
irods-csi-driver:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o bin/irods-csi-driver ./cmd/

.PHONY: driver_build
driver_build:
	docker build -t $(CSI_DRIVER_BUILD_IMAGE):latest -f $(CSI_DRIVER_BUILD_DOCKERFILE) .

.PHONY: image
image: driver_build
	docker build -t $(CSI_DRIVER_POOL_IMAGE):latest -f $(CSI_DRIVER_POOL_DOCKERFILE) .
	docker build -t $(CSI_DRIVER_IMAGE):latest -f $(CSI_DRIVER_DOCKERFILE) .

.PHONY: image-clean
image-clean: 
	docker rmi -f cyverse/irods-csi-driver:latest cyverse/irods-csi-driver:$(VERSION) cyverse/irods-csi-driver-pool:latest cyverse/irods-csi-driver-pool:$(VERSION) irods_csi_driver_build irods_csi_driver_pool_build irods_fuse_client_build -f


.PHONY: push
push: image
	docker push $(CSI_DRIVER_POOL_IMAGE):latest
	docker push $(CSI_DRIVER_IMAGE):latest

.PHONY: image-release
image-release:
	docker build -t $(CSI_DRIVER_POOL_IMAGE):$(VERSION) -f $(CSI_DRIVER_POOL_DOCKERFILE) .
	docker build -t $(CSI_DRIVER_IMAGE):$(VERSION) -f $(CSI_DRIVER_DOCKERFILE) .

.PHONY: push-release
push-release:
	docker push $(CSI_DRIVER_POOL_IMAGE):$(VERSION)
	docker push $(CSI_DRIVER_IMAGE):$(VERSION)

.PHONY: helm
helm:
	helm lint helm && helm package helm
