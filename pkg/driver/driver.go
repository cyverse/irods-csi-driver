package driver

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/commons"
	irodsfsd_client "github.com/cyverse/irodsfsd/client"
)

const irodsfsdOperationTimeout = time.Minute

// Driver object contains configuration parameters, grpc server and mounter
type Driver struct {
	config *commons.Config
	mode   commons.DriverMode

	server         *grpc.Server
	irodsfsdClient *irodsfsd_client.MountServiceClient
	mounter        Mounter
	secrets        map[string]string
}

// NewDriver returns new driver
func NewDriver(conf *commons.Config) (*Driver, error) {
	driver := &Driver{
		config:  conf,
		mode:    conf.GetDriverMode(),
		mounter: NewNodeMounter(),
		secrets: make(map[string]string),
	}
	if driver.mode == commons.NodeDriverMode {
		irodsfsdEndpoint := conf.GetIRODSFSDServiceEndpoint()
		irodsfsdClient := irodsfsd_client.NewMountServiceClient(
			irodsfsdEndpoint,
			irodsfsdOperationTimeout,
			true,
			nil,
		)
		if err := irodsfsdClient.Connect(); err != nil {
			return nil, fmt.Errorf("connect to irodsfsd at %q: %w", irodsfsdEndpoint, err)
		}
		readyContext, cancel := context.WithTimeout(context.Background(), irodsfsdOperationTimeout)
		defer cancel()
		if err := irodsfsdClient.Ready(readyContext); err != nil {
			irodsfsdClient.Disconnect()
			return nil, fmt.Errorf("verify irodsfsd at %q: %w", irodsfsdEndpoint, err)
		}
		driver.irodsfsdClient = irodsfsdClient
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

	return driver, nil
}

// Run runs the driver service
func (driver *Driver) Run() error {
	scheme, addr, err := commons.ParseServiceEndpoint(driver.config.GetServiceEndpoint())
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
	if driver.mode == commons.ControllerDriverMode {
		csi.RegisterControllerServer(driver.server, driver)
	} else {
		csi.RegisterNodeServer(driver.server, driver)
	}

	klog.V(3).Infof("Listening for connections on address %q", addr)
	return driver.server.Serve(listener)
}

// Stop stops the driver service
func (driver *Driver) Stop() {
	klog.V(3).Infof("Stopping server")
	if driver.server != nil {
		driver.server.Stop()
	}
	if driver.irodsfsdClient != nil {
		driver.irodsfsdClient.Disconnect()
	}
}
