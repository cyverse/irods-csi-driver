package driver

import (
	"context"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/client"
	"github.com/cyverse/irods-csi-driver/pkg/commons"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var nodeCaps = []csi.NodeServiceCapability_RPC_Type{
	csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
}

func (driver *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	volID := req.GetVolumeId()
	stagingPath := req.GetStagingTargetPath()
	if err := validateNodeMountRequest(volID, stagingPath, req.GetVolumeCapability()); err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	alreadyMounted, err := driver.ensureMountTarget(stagingPath)
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}
	if alreadyMounted {
		return &csi.NodeStageVolumeResponse{}, nil
	}

	configs := commons.MergeConfig(driver.config, driver.secrets, req.GetSecrets(), req.GetVolumeContext())
	mountOptions := GetMountOptions(req.GetVolumeCapability().GetMount(), req.GetVolumeCapability().GetAccessMode())
	if err := client.MountClient(driver.irodsfsdClient, volID, configs, mountOptions, stagingPath); err != nil {
		return nil, err
	}
	klog.V(5).Infof("NodeStageVolume: mounted %q at %q", volID, stagingPath)
	return &csi.NodeStageVolumeResponse{}, nil
}

func (driver *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	if err := validateNodeMountRequest(volID, targetPath, req.GetVolumeCapability()); err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}
	stagingPath := req.GetStagingTargetPath()
	if stagingPath == "" {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "staging target path not provided")
	}
	alreadyMounted, err := driver.ensureMountTarget(targetPath)
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}
	if alreadyMounted {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	mountOptions := GetMountOptions(req.GetVolumeCapability().GetMount(), req.GetVolumeCapability().GetAccessMode())
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}
	if err := MountBind(driver.mounter, stagingPath, mountOptions, targetPath); err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}
	klog.V(5).Infof("NodePublishVolume: bind mounted %q at %q", volID, targetPath)
	return &csi.NodePublishVolumeResponse{}, nil
}

func (driver *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID not provided")
	}
	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path not provided")
	}

	notMountPoint, err := driver.mounter.IsLikelyNotMountPoint(targetPath)
	if os.IsNotExist(err) {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}
	if err != nil {
		commons.IncreaseCounterForVolumeUnmountFailures()
		return nil, status.Errorf(codes.Internal, "check target path %q: %v", targetPath, err)
	}
	if !notMountPoint {
		if err := driver.mounter.Unmount(targetPath); err != nil {
			commons.IncreaseCounterForVolumeUnmountFailures()
			return nil, status.Errorf(codes.Internal, "unmount target path %q: %v", targetPath, err)
		}
		commons.IncreaseCounterForVolumeUnmount()
		commons.DecreaseCounterForActiveVolumeMount()
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return nil, status.Errorf(codes.Internal, "remove target path %q: %v", targetPath, err)
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (driver *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	volID := req.GetVolumeId()
	if volID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID not provided")
	}
	if req.GetStagingTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path not provided")
	}
	if err := client.UnmountVolume(driver.irodsfsdClient, volID); err != nil {
		return nil, err
	}
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (driver *Driver) ensureMountTarget(targetPath string) (bool, error) {
	exists, err := PathExists(targetPath)
	if err != nil {
		return false, status.Errorf(codes.Internal, "check target path %q: %v", targetPath, err)
	}
	if !exists {
		if err := MakeDir(targetPath); err != nil {
			return false, status.Errorf(codes.Internal, "create target path %q: %v", targetPath, err)
		}
	}
	notMountPoint, err := driver.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		return false, status.Errorf(codes.Internal, "check mount point %q: %v", targetPath, err)
	}
	return !notMountPoint, nil
}

func validateNodeMountRequest(volID, targetPath string, capability *csi.VolumeCapability) error {
	if volID == "" {
		return status.Error(codes.InvalidArgument, "volume ID not provided")
	}
	if targetPath == "" {
		return status.Error(codes.InvalidArgument, "target path not provided")
	}
	if capability == nil {
		return status.Error(codes.InvalidArgument, "volume capability not provided")
	}
	if !isValidVolumeCapabilities([]*csi.VolumeCapability{capability}) {
		return status.Error(codes.InvalidArgument, "volume capability not supported")
	}
	return nil
}

func (driver *Driver) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
func (driver *Driver) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (driver *Driver) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	capabilities := make([]*csi.NodeServiceCapability, 0, len(nodeCaps))
	for _, capability := range nodeCaps {
		capabilities = append(capabilities, &csi.NodeServiceCapability{Type: &csi.NodeServiceCapability_Rpc{Rpc: &csi.NodeServiceCapability_RPC{Type: capability}}})
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: capabilities}, nil
}

func (driver *Driver) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: driver.config.NodeID}, nil
}
