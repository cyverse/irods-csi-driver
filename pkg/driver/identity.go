package driver

import (
	"context"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/commons"
	"github.com/golang/protobuf/ptypes/wrappers"
	"k8s.io/klog"
)

const irodsfsdProbeTimeout = 5 * time.Second

// GetPluginInfo returns plugin info
func (driver *Driver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	resp := &csi.GetPluginInfoResponse{
		Name:          commons.DriverName,
		VendorVersion: commons.GetDriverVersion(),
	}

	return resp, nil
}

// GetPluginCapabilities returns plugin capabilities
func (driver *Driver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	resp := &csi.GetPluginCapabilitiesResponse{}
	if driver.mode == commons.ControllerDriverMode {
		resp.Capabilities = []*csi.PluginCapability{{
			Type: &csi.PluginCapability_Service_{
				Service: &csi.PluginCapability_Service{Type: csi.PluginCapability_Service_CONTROLLER_SERVICE},
			},
		}}
	}

	return resp, nil
}

// Probe returns probe response
func (driver *Driver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	ready := driver.mode == commons.ControllerDriverMode
	if driver.mode == commons.NodeDriverMode && driver.irodsfsdClient != nil {
		probeContext, cancel := context.WithTimeout(ctx, irodsfsdProbeTimeout)
		defer cancel()

		if err := driver.irodsfsdClient.Ready(probeContext); err != nil {
			klog.V(2).Infof("irodsfsd is not ready: %v", err)
		} else {
			ready = true
		}
	}

	return &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: ready}}, nil
}
