/*
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

/*
Copyright 2020 CyVerse
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
)

const (
	driverName = "irods.csi.cyverse.org"
)

var (
	volumeCaps = []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}
)

// Driver object contains configuration parameters, grpc server and mounter
type Driver struct {
	config *Config

	server  *grpc.Server
	mounter Mounter
	secrets map[string]string
}

// NewDriver returns new driver
func NewDriver(conf *Config) *Driver {
	return &Driver{
		config:  conf,
		mounter: newNodeMounter(),
	}
}

// Run runs the driver service
func (driver *Driver) Run() error {
	scheme, addr, err := ParseEndpoint(driver.config.Endpoint)
	if err != nil {
		return err
	}

	driver.secrets = make(map[string]string)
	secrets, err := ReadIRODSSecrets(driver.config.SecretPath)
	if err == nil {
		// if there's no secrets, it returns error, so we ignore
		// otherwise, copy
		for k, v := range secrets {
			driver.secrets[k] = v
		}
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

	klog.Infof("Listening for connections on address: %#v", listener.Addr())
	return driver.server.Serve(listener)
}

// Stop stops the driver service
func (driver *Driver) Stop() {
	klog.Infof("Stopping server")
	driver.server.Stop()
}

// isValidVolumeCapabilities checks validity of volume capabilities
func (driver *Driver) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, m := range volumeCaps {
			if m == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
		}
	}
	return foundAll
}
