# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

PKG=github.com/cyverse/irods-csi-driver
CSI_DRIVER_BUILD_IMAGE=irods_csi_driver_build
CSI_DRIVER_BUILD_DOCKERFILE=deploy/image/irods_csi_driver_build.dockerfile
FUSE_CLIENT_BUILD_IMAGE=irods_fuse_client_build
FUSE_CLIENT_BUILD_DOCKERFILE=deploy/image/irods_fuse_build.dockerfile
CSI_DRIVER_IMAGE?=cyverse/irods-csi-driver
CSI_DRIVER_DOCKERFILE=deploy/image/irods_csi_driver_image.dockerfile
VERSION=v0.2.0
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X ${PKG}/pkg/driver.driverVersion=${VERSION} -X ${PKG}/pkg/driver.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/driver.buildDate=${BUILD_DATE}"
GO111MODULE=on
GOPROXY=direct
GOPATH=$(shell go env GOPATH)

.EXPORT_ALL_VARIABLES:

.PHONY: irods-csi-driver
irods-csi-driver:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o bin/irods-csi-driver ./cmd/

#.PHONY: verify
#verify:
#	./hack/verify-all

#.PHONY: test
#test:
#	go test -v -race $$(go list ./pkg/... | grep -v /driver)
#	# TODO stop skipping controller tests when controller is implemented
#	go test -v -race ./pkg/driver/... -ginkgo.skip='\[Controller.Server\]'

#.PHONY: test-e2e
#test-e2e:
#	go get github.com/aws/aws-k8s-tester/e2e/tester/cmd/k8s-e2e-tester@master
#	TESTCONFIG=./tester/e2e-test-config.yaml ${GOPATH}/bin/k8s-e2e-tester

.PHONY: fuse_build
fuse_build:
	docker build -t $(FUSE_CLIENT_BUILD_IMAGE):latest -f $(FUSE_CLIENT_BUILD_DOCKERFILE) .

.PHONY: driver_build
driver_build:
	docker build -t $(CSI_DRIVER_BUILD_IMAGE):latest -f $(CSI_DRIVER_BUILD_DOCKERFILE) .

.PHONY: image
image: fuse_build driver_build
	docker build -t $(CSI_DRIVER_IMAGE):latest -f $(CSI_DRIVER_DOCKERFILE) .

.PHONY: push
push: image
	docker push $(CSI_DRIVER_IMAGE):latest

.PHONY: image-release
image-release:
	docker build -t $(CSI_DRIVER_IMAGE):$(VERSION) -f $(CSI_DRIVER_DOCKERFILE) .

.PHONY: push-release
push-release:
	docker push $(CSI_DRIVER_IMAGE):$(VERSION)
