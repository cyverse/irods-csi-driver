/*
Following functions or objects are from the code under APL2 License.
- CreateVolume
- DeleteVolume
- ControllerGetCapabilities
- ValidateVolumeCapabilities
Original code:
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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/client"
	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"github.com/cyverse/irods-csi-driver/pkg/metrics"
	"github.com/cyverse/irods-csi-driver/pkg/volumeinfo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var (
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}

	// defaultVolumeSize specifies default volume size in Bytes
	defaultVolumeSize int64 = 100 * 1024 * 1024 * 1024
)

// CreateVolume handles persistent volume creation event
func (driver *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// volume name is created by CO for idempotency
	volName := req.GetName()
	if len(volName) == 0 {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}
	volID := generateVolumeID(volName)

	klog.V(4).Infof("CreateVolume: volumeName(%#v)", volName)

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if !isValidVolumeCapabilities(volCaps) {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported")
	}

	capRange := req.GetCapacityRange()
	volCapacity := defaultVolumeSize
	if capRange != nil {
		volCapacity = capRange.GetRequiredBytes()
	}

	// create a new volume
	// merge params
	configs := mergeConfig(driver.config, driver.secrets, req.GetSecrets(), req.GetParameters())

	///////////////////////////////////////////////////////////
	// We only support irodsfs for dynamic volume provisioning
	///////////////////////////////////////////////////////////
	irodsClientType := client.GetClientType(configs)
	if irodsClientType != client.IrodsFuseClientType {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Errorf(codes.InvalidArgument, "unsupported driver type - %v", irodsClientType)
	}

	// make controller config
	controllerConfig, err := MakeControllerConfig(volName, configs)
	if err != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	// set path
	configs["path"] = controllerConfig.VolumePath

	// get iRODS connection info
	irodsConnectionInfo, err := irods.GetConnectionInfo(configs)
	if err != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	// generate path
	if !controllerConfig.NotCreateVolumeDir {
		// create
		klog.V(5).Infof("Creating a volume dir %s", controllerConfig.VolumePath)
		err = irods.Mkdir(irodsConnectionInfo, controllerConfig.VolumePath)
		if err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, status.Errorf(codes.Internal, "Could not create a volume dir %s : %v", controllerConfig.VolumePath, err)
		}
	}

	// do not allow anonymous access for dynamic volume provisioning since it creates a new empty volume
	if irodsConnectionInfo.IsAnonymousUser() {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Argument user must be a non-anonymous user")
	}

	// copy config values to volContext, to be used in node
	volContext := make(map[string]string)
	for k, v := range req.GetParameters() {
		volContext[k] = v
	}
	volContext["path"] = controllerConfig.VolumePath

	// tell this volume is created via dynamic volume provisioning
	setDynamicVolumeProvisioningMode(volContext)

	// create a controller volume (for dynamic volume provisioning)
	controllerVolume := &volumeinfo.ControllerVolume{
		ID:             volID,
		Name:           volName,
		RootPath:       controllerConfig.VolumeRootPath,
		Path:           controllerConfig.VolumePath,
		ConnectionInfo: irodsConnectionInfo,
		RetainData:     controllerConfig.RetainData,
	}
	driver.controllerVolumeManager.Put(controllerVolume)

	volume := &csi.Volume{
		VolumeId:      volID,
		CapacityBytes: volCapacity,
		VolumeContext: volContext,
	}

	return &csi.CreateVolumeResponse{Volume: volume}, nil
}

// DeleteVolume handles persistent volume deletion event
func (driver *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("DeleteVolume: volumeId (%#v)", volID)

	controllerVolume := driver.controllerVolumeManager.Pop(volID)
	if controllerVolume == nil {
		// orphant
		klog.V(4).Infof("DeleteVolume: cannot find a volume with id (%v)", volID)
		// ignore this error
		return &csi.DeleteVolumeResponse{}, nil
	}

	if !controllerVolume.RetainData {
		klog.V(5).Infof("Deleting a volume dir %s", controllerVolume.Path)
		err := irods.Rmdir(controllerVolume.ConnectionInfo, controllerVolume.Path)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not delete a volume dir %s : %v", controllerVolume.Path, err)
		}
	}

	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume handles persistent volume publish event in controller service
func (driver *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerUnpublishVolume handles persistent volume unpublish event in controller service
func (driver *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities returns capabilities
func (driver *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(4).Infof("ControllerGetCapabilities: called with args %#v", req)

	var caps []*csi.ControllerServiceCapability
	for _, cap := range controllerCaps {
		c := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

// GetCapacity returns volume capacity
func (driver *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	klog.V(4).Infof("GetCapacity: called with args %#v", req)
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes returns a list of volumes created
func (driver *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.V(4).Infof("ListVolumes: called with args %#v", req)
	return nil, status.Error(codes.Unimplemented, "")
}

// ValidateVolumeCapabilities checks validity of volume capabilities
func (driver *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	klog.V(4).Infof("ValidateVolumeCapabilities: called with args %#v", req)
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	confirmed := isValidVolumeCapabilities(volCaps)
	if confirmed {
		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				// TODO if volume context is provided, should validate it too
				// VolumeContext:      req.GetVolumeContext(),
				VolumeCapabilities: volCaps,
				// TODO if parameters are provided, should validate them too
				// Parameters:      req.GetParameters(),
			},
		}, nil
	}

	return &csi.ValidateVolumeCapabilitiesResponse{}, nil
}

// CreateSnapshot creates a snapshot of a volume
func (driver *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteSnapshot deletes a snapshot of a volume
func (driver *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListSnapshots returns a list of snapshots
func (driver *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerExpandVolume expands a volume
func (driver *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
