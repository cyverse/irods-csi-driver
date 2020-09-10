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
	volID := generateVolumeID(volName)

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

	secrets := make(map[string]string)
	for k, v := range driver.secrets {
		secrets[k] = v
	}

	for k, v := range volSecrets {
		secrets[k] = v
	}

	irodsClient := ExtractIRODSClientType(volParams, secrets, FuseType)
	if irodsClient != FuseType {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported driver type - %v", irodsClient)
	}

	// check security flags
	enforceProxyAccess := false
	proxyUser := ""
	for k, v := range driver.secrets {
		if strings.ToLower(k) == "enforceproxyaccess" {
			enforce, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			enforceProxyAccess = enforce
		}

		if strings.ToLower(k) == "user" {
			proxyUser = v
		}
	}

	irodsConn, err := ExtractIRODSConnection(volParams, secrets)
	if err != nil {
		return nil, err
	}

	if enforceProxyAccess {
		if proxyUser == irodsConn.User {
			// same proxy user
			// enforce clientUser
			if len(irodsConn.ClientUser) == 0 {
				return nil, status.Error(codes.InvalidArgument, "Argument clientUser must be given")
			}

			if irodsConn.User == irodsConn.ClientUser {
				return nil, status.Errorf(codes.InvalidArgument, "Argument clientUser cannot be the same as user - user %s, clientUser %s", irodsConn.User, irodsConn.ClientUser)
			}
		} else {
			// replaced user
			// static volume provisioning takes user argument from pv
			// this is okay
		}
	}

	volContext := make(map[string]string)
	volRetain := false
	volCreate := true
	volPath := ""
	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "volumerootpath":
			if !filepath.IsAbs(v) {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}
			volRootPath = strings.TrimRight(v, "/")
		case "retaindata":
			retain, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			volRetain = retain
		case "novolumedir":
			novolumedir, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			volCreate = !novolumedir
		}
		// do not copy secret params
	}

	for k, v := range volParams {
		switch strings.ToLower(k) {
		case "volumerootpath":
			if !filepath.IsAbs(v) {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}
			volRootPath = strings.TrimRight(v, "/")
		case "retaindata":
			retain, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			volRetain = retain
		case "novolumedir":
			novolumedir, err := strconv.ParseBool(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			volCreate = !novolumedir
		}
		// copy all params
		volContext[k] = v
	}

	if len(volRootPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument volumeRootPath is not provided")
	}

	// generate path
	if volCreate {
		volPath = fmt.Sprintf("%s/%s", volRootPath, volName)

		klog.V(5).Infof("Creating a volume dir %s", volPath)
		err = IRODSMkdir(irodsConn, volPath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not create a volume dir %s : %v", volPath, err)
		}
	} else {
		volPath = volRootPath
		// in this case, we should retain data because the mounted path may have files
		// we should not delete these old files when the pvc is deleted
		volRetain = true
	}

	volContext["path"] = volPath

	// create a irods volume and put it to manager
	irodsVolume := NewIRODSVolume(volID, volName, volRootPath, volPath, irodsConn, volRetain)
	PutIRODSVolume(irodsVolume)

	volume := &csi.Volume{
		VolumeId:      volID,
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
			return nil, status.Errorf(codes.Internal, "Could not delete a volume dir %s : %v", irodsVolume.Path, err)
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

// generateVolumeID generates volume id from volume name
func generateVolumeID(volName string) string {
	//uuid := uuid.New()
	//return fmt.Sprintf("volid-%s", uuid.String())
	return volName
}
