/*
Following functions or objects are from the code under APL2 License.
- NodeStageVolume
- NodePublishVolume
- NodeUnpublishVolume
- NodeUnstageVolume
- NodeGetCapabilities
- NodeGetInfo
Original code:
- https://github.com/kubernetes-sigs/aws-efs-csi-driver/blob/master/pkg/driver/node.go
- https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/master/pkg/driver/node.go


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
	"fmt"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/client"
	"github.com/cyverse/irods-csi-driver/pkg/metrics"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"github.com/cyverse/irods-csi-driver/pkg/volumeinfo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var (
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}
)

// NodeStageVolume handles persistent volume stage event in node service
func (driver *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodeStageVolume: volumeId (%#v)", volID)

	if !isDynamicVolumeProvisioningMode(req.GetVolumeContext()) {
		// if it is static volume provisioning, just return quick.
		// nothing to do.
		nodeVolume := &volumeinfo.NodeVolume{
			ID:                        volID,
			StagingMountPath:          "",
			MountPath:                 "",
			DynamicVolumeProvisioning: false,
			StageVolume:               true,
		}
		err := driver.nodeVolumeManager.Put(nodeVolume)
		if err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, err
		}

		return &csi.NodeStageVolumeResponse{}, nil
	}

	// only for dynamic volume provisioning mode
	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	mountOptions := mounter.GetMountOptions(volCap.GetMount())

	pathExist, pathExistErr := mounter.PathExists(targetPath)
	if pathExistErr != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.Internal, pathExistErr.Error())
	}

	if !pathExist {
		klog.V(5).Infof("NodeStageVolume: creating dir %s", targetPath)
		if err := mounter.MakeDir(targetPath); err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", targetPath, err)
		}
	}

	notMountPoint, err := driver.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !notMountPoint {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Errorf(codes.Internal, "Staging target path %s is already mounted", targetPath)
	}

	// merge params
	configs := mergeConfig(driver.config, driver.secrets, req.GetSecrets(), req.GetVolumeContext())
	klog.V(5).Infof("NodeStageVolume: mounting %s", targetPath)

	// mount
	err = client.MountClient(driver.mounter, configs, mountOptions, targetPath)
	if err != nil {
		return nil, err
	}

	klog.V(5).Infof("NodeStageVolume: %s was mounted", targetPath)

	nodeVolume := &volumeinfo.NodeVolume{
		ID:                        volID,
		StagingMountPath:          targetPath,
		MountPath:                 "",
		DynamicVolumeProvisioning: true,
		StageVolume:               true,
	}

	err = driver.nodeVolumeManager.Put(nodeVolume)
	if err != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// NodePublishVolume handles persistent volume publish event in node service
func (driver *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodePublishVolume: volumeId (%#v)", volID)

	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	mountOptions := mounter.GetMountOptions(volCap.GetMount())
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	pathExist, pathExistErr := mounter.PathExists(targetPath)
	if pathExistErr != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.Internal, pathExistErr.Error())
	}

	if !pathExist {
		klog.V(5).Infof("NodePublishVolume: creating dir %s", targetPath)
		if err := mounter.MakeDir(targetPath); err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", targetPath, err)
		}
	}

	notMountPoint, err := driver.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !notMountPoint {
		metrics.IncreaseCounterForVolumeMountFailures()
		return nil, status.Errorf(codes.Internal, "Staging target path %s is already mounted", targetPath)
	}

	if isDynamicVolumeProvisioningMode(req.GetVolumeContext()) {
		// dynamic volume provisioning
		// bind mount
		stagingTargetPath := req.GetStagingTargetPath()
		if len(stagingTargetPath) == 0 {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
		}

		klog.V(5).Infof("NodePublishVolume: bind mounting %s", targetPath)
		if err := mounter.MountBind(driver.mounter, stagingTargetPath, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, err
		}

		// update node volume info
		nodeVolume, err := driver.nodeVolumeManager.Pop(volID)
		if err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, err
		}

		if nodeVolume == nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, status.Errorf(codes.InvalidArgument, "Unable to find node volume %s", volID)
		}

		nodeVolume.MountPath = targetPath
		err = driver.nodeVolumeManager.Put(nodeVolume)
		if err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
	} else {
		// static volume provisioning
		// merge params
		configs := mergeConfig(driver.config, driver.secrets, req.GetSecrets(), req.GetVolumeContext())

		// mount
		klog.V(5).Infof("NodePublishVolume: mounting %s", targetPath)
		err = client.MountClient(driver.mounter, configs, mountOptions, targetPath)
		if err != nil {
			return nil, err
		}

		// update node volume info if exists
		nodeVolume, err := driver.nodeVolumeManager.Pop(volID)
		if err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, err
		}

		if nodeVolume == nil {
			nodeVolume = &volumeinfo.NodeVolume{
				ID:                        volID,
				StagingMountPath:          "",
				MountPath:                 targetPath,
				DynamicVolumeProvisioning: false,
				StageVolume:               false,
			}
		} else {
			nodeVolume.MountPath = targetPath
		}

		err = driver.nodeVolumeManager.Put(nodeVolume)
		if err != nil {
			metrics.IncreaseCounterForVolumeMountFailures()
			return nil, err
		}
	}

	klog.V(5).Infof("NodePublishVolume: %s was mounted", targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume handles persistent volume unpublish event in node service
func (driver *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodeUnpublishVolume: volumeId (%#v)", volID)

	nodeVolume := driver.nodeVolumeManager.Get(volID)
	if nodeVolume == nil {
		klog.Errorf("Unable to find node volume %s in the node volume manager, but we continue anyway", volID)
	} else {
		if !nodeVolume.StageVolume {
			// if the volume is added at NodeStageVolume, delete here
			_, err := driver.nodeVolumeManager.Pop(volID)
			if err != nil {
				return nil, err
			}
		}
	}

	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	_, refCount, err := driver.mounter.GetDeviceName(targetPath)
	if err != nil {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		msg := fmt.Sprintf("failed to check if volume is mounted: %v", err)
		return nil, status.Error(codes.Internal, msg)
	}

	// From the spec: If the volume corresponding to the volume_id
	// is not staged to the staging_target_path, the Plugin MUST
	// reply 0 OK.
	if refCount == 0 {
		klog.V(5).Infof("NodeUnpublishVolume: %s target not mounted", targetPath)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	klog.V(5).Infof("NodeUnpublishVolume: unmounting %s", targetPath)
	// unmount
	err = driver.mounter.UnmountForcefully(targetPath)
	if err != nil {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		return nil, status.Errorf(codes.Internal, "Failed to unmount %q: %v", targetPath, err)
	}
	klog.V(5).Infof("NodeUnpublishVolume: %s unmounted", targetPath)

	metrics.IncreaseCounterForVolumeUnmount()
	metrics.DecreaseCounterForActiveVolumeMount()

	err = os.Remove(targetPath)
	if err != nil && !os.IsNotExist(err) {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeUnstageVolume handles persistent volume unstage event in node service
func (driver *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodeUnstageVolume: volumeId (%#v)", volID)

	nodeVolume := driver.nodeVolumeManager.Get(volID)
	if nodeVolume == nil {
		klog.V(5).Infof("Unable to find node volume %s in the node volume manager, but we continue anyway", volID)
	} else {
		// delete here
		_, err := driver.nodeVolumeManager.Pop(volID)
		if err != nil {
			return nil, err
		}

		if !nodeVolume.DynamicVolumeProvisioning {
			// nothing to do for StaticCVolumeProvisioning
			return &csi.NodeUnstageVolumeResponse{}, nil
		}
	}

	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	_, refCount, err := driver.mounter.GetDeviceName(targetPath)
	if err != nil {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		msg := fmt.Sprintf("failed to check if volume is mounted: %v", err)
		return nil, status.Error(codes.Internal, msg)
	}

	// From the spec: If the volume corresponding to the volume_id
	// is not staged to the staging_target_path, the Plugin MUST
	// reply 0 OK.
	if refCount == 0 {
		klog.V(5).Infof("NodeUnstageVolume: %s target not mounted", targetPath)
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	klog.V(5).Infof("NodeUnstageVolume: unmounting %s", targetPath)
	err = driver.mounter.UnmountForcefully(targetPath)
	if err != nil {
		metrics.IncreaseCounterForVolumeUnmountFailures()
		return nil, status.Errorf(codes.Internal, "Failed to unmount %q: %v", targetPath, err)
	}
	klog.V(5).Infof("NodeUnstageVolume: %s unmounted", targetPath)

	metrics.IncreaseCounterForVolumeUnmount()
	metrics.DecreaseCounterForActiveVolumeMount()

	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodeGetVolumeStats returns volume stats
func (driver *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeExpandVolume expands volume
func (driver *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeGetCapabilities returns capabilities
func (driver *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(4).Infof("NodeGetCapabilities: called with args %+v", req)
	var caps []*csi.NodeServiceCapability
	for _, cap := range nodeCaps {
		c := &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

// NodeGetInfo returns node info
func (driver *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.V(4).Infof("NodeGetInfo: called with args %+v", req)

	return &csi.NodeGetInfoResponse{
		NodeId: driver.config.NodeID,
	}, nil
}
