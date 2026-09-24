PKG=github.com/cyverse/irods-csi-driver
CSI_DRIVER_IMAGE?=cyverse/irods-csi-driver
CSI_DRIVER_DOCKERFILE=deploy/image/irods_csi_driver_image.dockerfile
VERSION=v0.12.1
GOOS?=linux
GOARCH?=$(shell go env GOARCH)
PLATFORM?=$(GOOS)/$(GOARCH)
PLATFORMS?=linux/amd64,linux/arm64
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X ${PKG}/pkg/commons.driverVersion=${VERSION} -X ${PKG}/pkg/commons.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/commons.buildDate=${BUILD_DATE}"
GO111MODULE=on
GOPROXY=direct
GOPATH=$(shell go env GOPATH)

.EXPORT_ALL_VARIABLES:

.PHONY: irods-csi-driver
irods-csi-driver:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags ${LDFLAGS} -o bin/irods-csi-driver ./cmd/

.PHONY: image
image:
	docker build --platform=$(PLATFORM) --build-arg VERSION=$(VERSION) -t $(CSI_DRIVER_IMAGE):latest -f $(CSI_DRIVER_DOCKERFILE) .

.PHONY: release
release:
	docker buildx build --platform=$(PLATFORMS) --push --build-arg VERSION=$(VERSION) -t $(CSI_DRIVER_IMAGE):$(VERSION) -t $(CSI_DRIVER_IMAGE):latest -f $(CSI_DRIVER_DOCKERFILE) .
	helm lint helm && helm package helm

.PHONY: helm
helm:
	helm lint helm && helm package helm
