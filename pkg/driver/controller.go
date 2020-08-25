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
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var (
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}

	// DefaultVolumeSize specifies default volume size in Bytes
	DefaultVolumeSize int64 = 100 * 1024 * 1024 * 1024
)

// CreateVolume handles persistent volume creation event
func (driver *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	//klog.V(4).Infof("CreateVolume: called with args %#v", req)

	// volume name is created by CO for idempotency
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}

	klog.V(4).Infof("CreateVolume: volumeName(%#v)", volName)

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if !driver.isValidVolumeCapabilities(volCaps) {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported")
	}

	capRange := req.GetCapacityRange()
	volCapacity := DefaultVolumeSize
	if capRange != nil {
		volCapacity = capRange.GetRequiredBytes()
	}

	// create a new volume
	// need to provide idempotency
	volParams := req.GetParameters()
	volSecrets := req.GetSecrets()
	volRootPath := ""

	irodsClient := ExtractIRODSClientType(volParams, FuseType)
	if irodsClient != FuseType {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported driver type - %v", irodsClient)
	}

	irodsConn, err := ExtractIRODSConnection(volParams, volSecrets)
	if err != nil {
		return nil, err
	}

	volContext := make(map[string]string)
	volRetain := false
	for k, v := range volParams {
		switch strings.ToLower(k) {
		case "volumerootpath":
			if !filepath.IsAbs(v) {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Volume parameter property %q must be an absolute path", k))
			}
			volRootPath = strings.TrimRight(v, "/")
		case "retaindata":
			retain, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Argument %q must be a boolean value - %s", k, err))
			}
			volRetain = retain
		}
		// copy all params
		volContext[k] = v
	}

	if len(volRootPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume parameter 'volumeRootPath' not provided")
	}

	// generate path
	volPath := fmt.Sprintf("%s/%s", volRootPath, volName)

	klog.V(5).Infof("Creating a volume dir %s", volPath)
	err = IRODSMkdir(irodsConn, volPath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Could not create a volume dir %s : %v", volPath, err))
	}

	volContext["path"] = volPath

	// create a irods volume and put it to manager
	irodsVolume := NewIRODSVolume(volName, volName, volRootPath, volPath, irodsConn, volRetain)
	PutIRODSVolume(irodsVolume)

	volume := &csi.Volume{
		VolumeId:      volName,
		CapacityBytes: volCapacity,
		VolumeContext: volContext,
	}

	return &csi.CreateVolumeResponse{Volume: volume}, nil
}

// DeleteVolume handles persistent volume deletion event
func (driver *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	//klog.V(4).Infof("DeleteVolume: called with args: %#v", req)

	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("DeleteVolume: volumeId (%#v)", volID)

	irodsVolume := PopIRODSVolume(volID)
	if irodsVolume == nil {
		// orphant
		klog.V(4).Infof("DeleteVolume: cannot find a volume with id (%v)", volID)
		// ignore this error
		return &csi.DeleteVolumeResponse{}, nil
	}

	if !irodsVolume.RetainData {
		klog.V(5).Infof("Deleting a volume dir %s", irodsVolume.Path)
		err := IRODSRmdir(irodsVolume.Connection, irodsVolume.Path)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Could not delete a volume dir %s : %v", irodsVolume.Path, err))
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

	confirmed := driver.isValidVolumeCapabilities(volCaps)
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
