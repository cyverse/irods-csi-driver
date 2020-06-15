package driver

import (
	"k8s.io/klog"
    "google.golang.org/grpc"

    "github.com/iychoi/irods-csi-driver/pkg/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

// Driver contains the default identity,node and controller struct
type Driver struct {
    config *util.Config

    server *grpc.Server

    mounter Mounter
}

// NewDriver returns new ceph driver
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
