/*
Following functions or objects are from the code under APL2 License.
- Driver
- NewDriver
- Run
- Stop
- isValidVolumeCapabilities
Original code:
- https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/master/pkg/driver/driver.go
- https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/master/pkg/driver/controller.go


Copyright 2019 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"k8s.io/klog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/common"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"github.com/cyverse/irods-csi-driver/pkg/volumeinfo"
)

var (
	volumeCaps = []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}
)

// Driver object contains configuration parameters, grpc server and mounter
type Driver struct {
	config *common.Config

	server  *grpc.Server
	mounter mounter.Mounter
	secrets map[string]string

	controllerVolumeManager *volumeinfo.ControllerVolumeManager
	nodeVolumeManager       *volumeinfo.NodeVolumeManager
}

// NewDriver returns new driver
func NewDriver(conf *common.Config) (*Driver, error) {
	driver := &Driver{
		config:  conf,
		mounter: mounter.NewNodeMounter(),
		secrets: make(map[string]string),

		controllerVolumeManager: nil,
		nodeVolumeManager:       nil,
	}

	// update secrets
	driver.secrets = make(map[string]string)
	secrets, err := readSecrets(driver.config.SecretPath)
	if err == nil {
		// if there's no secrets, it returns error, so we ignore
		// otherwise, copy
		for k, v := range secrets {
			driver.secrets[k] = v
		}
	}

	volumeEncryptKey := "irodscsidriver_volume_2ce02bee-74ea-4b18-a440-472d9771f778"
	for k, v := range driver.secrets {
		if normalizeConfigKey(k) == "volumeencryptkey" {
			volumeEncryptKey = v
			break
		}
	}

	controllerVolumeManager, err := volumeinfo.NewControllerVolumeManager(volumeEncryptKey, conf.StoragePath)
	if err != nil {
		return nil, err
	}

	nodeVolumeManager, err := volumeinfo.NewNodeVolumeManager(volumeEncryptKey, conf.StoragePath)
	if err != nil {
		return nil, err
	}

	driver.controllerVolumeManager = controllerVolumeManager
	driver.nodeVolumeManager = nodeVolumeManager

	return driver, nil
}

// Run runs the driver service
func (driver *Driver) Run() error {
	scheme, addr, err := common.ParseEndpoint(driver.config.Endpoint)
	if err != nil {
		return err
	}

	listener, err := net.Listen(scheme, addr)
	if err != nil {
		return err
	}

	logErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			klog.Errorf("GRPC error: %v", err)
		}
		return resp, err
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logErr),
	}

	driver.server = grpc.NewServer(opts...)

	csi.RegisterIdentityServer(driver.server, driver)
	csi.RegisterControllerServer(driver.server, driver)
	csi.RegisterNodeServer(driver.server, driver)

	klog.V(3).Infof("Listening for connections on address: %#v", listener.Addr())
	return driver.server.Serve(listener)
}

// Stop stops the driver service
func (driver *Driver) Stop() {
	klog.V(3).Infof("Stopping server")
	driver.server.Stop()
}
