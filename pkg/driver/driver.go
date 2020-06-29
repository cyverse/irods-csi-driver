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

package driver

import (
	"context"
	"net"

	"k8s.io/klog"
    "google.golang.org/grpc"

    "github.com/cyverse/irods-csi-driver/pkg/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	driverName = "irods.csi.cyverse.org"
)

// contain the default identity, node and controller struct
type Driver struct {
    config *util.Config

    server *grpc.Server
	nodeID string //TODO

    mounter Mounter
}

// NewDriver returns new driver
func NewDriver(conf *util.Config) *Driver {
    return &Driver{
        config: conf,
        mounter: newNodeMounter(),
    }
}

func (driver *Driver) Run() error {
    scheme, addr, err := util.ParseEndpoint(driver.config.Endpoint)
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
	csi.RegisterNodeServer(driver.server, driver)

	klog.Infof("Listening for connections on address: %#v", listener.Addr())
	return driver.server.Serve(listener)
}
