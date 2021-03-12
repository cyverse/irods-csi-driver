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
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

var (
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME}
)

const (
	sensitiveArgsRemoved = "<masked>"
)

// NodeStageVolume handles persistent volume stage event in node service
func (driver *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodeStageVolume: volumeId (%#v)", volID)

	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !driver.isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	mountOptions := []string{}
	if m := volCap.GetMount(); m != nil {
		hasOption := func(options []string, opt string) bool {
			for _, o := range options {
				if o == opt {
					return true
				}
			}
			return false
		}
		for _, f := range m.MountFlags {
			if !hasOption(mountOptions, f) {
				mountOptions = append(mountOptions, f)
			}
		}
	}

	notMountPoint, err := driver.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !notMountPoint {
		return nil, status.Errorf(codes.Internal, "Staging target path %s is already mounted", targetPath)
	}

	volContext := req.GetVolumeContext()
	volSecrets := req.GetSecrets()

	secrets := make(map[string]string)
	for k, v := range driver.secrets {
		secrets[k] = v
	}

	for k, v := range volSecrets {
		secrets[k] = v
	}

	irodsClient := ExtractIRODSClientType(volContext, secrets, FuseType)

	switch irodsClient {
	case FuseType:
		klog.V(5).Infof("NodeStageVolume: mounting %s", irodsClient)
		if err := driver.mountFuse(volContext, secrets, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			return nil, err
		}
	case WebdavType:
		klog.V(5).Infof("NodeStageVolume: mounting %s", irodsClient)
		if err := driver.mountWebdav(volContext, secrets, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			return nil, err
		}
	case NfsType:
		klog.V(5).Infof("NodeStageVolume: mounting %s", irodsClient)
		if err := driver.mountNfs(volContext, secrets, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.Internal, "unknown driver type - %v", irodsClient)
	}

	klog.V(5).Infof("NodeStageVolume: %s was mounted", targetPath)
	return &csi.NodeStageVolumeResponse{}, nil
}

// NodePublishVolume handles persistent volume publish event in node service
func (driver *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	//klog.V(4).Infof("NodePublishVolume: called with args %+v", req)

	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodePublishVolume: volumeId (%#v)", volID)

	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !driver.isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	mountOptions := []string{}
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	if m := volCap.GetMount(); m != nil {
		hasOption := func(options []string, opt string) bool {
			for _, o := range options {
				if o == opt {
					return true
				}
			}
			return false
		}
		for _, f := range m.MountFlags {
			if !hasOption(mountOptions, f) {
				mountOptions = append(mountOptions, f)
			}
		}
	}

	pathExist, pathExistErr := PathExists(targetPath)
	if pathExistErr != nil {
		return nil, status.Error(codes.Internal, pathExistErr.Error())
	}

	if !pathExist {
		klog.V(5).Infof("NodePublishVolume: creating dir %s", targetPath)
		if err := MakeDir(targetPath); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", targetPath, err)
		}
	}

	notMountPoint, err := driver.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !notMountPoint {
		return nil, status.Errorf(codes.Internal, "Staging target path %s is already mounted", targetPath)
	}

	// bind mount
	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}

	klog.V(5).Infof("NodePublishVolume: mounting %s", "bind")
	if err := driver.mountBind(stagingTargetPath, mountOptions, targetPath); err != nil {
		os.Remove(targetPath)
		return nil, err
	}

	klog.V(5).Infof("NodePublishVolume: %s was mounted", targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume handles persistent volume unpublish event in node service
func (driver *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	//klog.V(4).Infof("NodeUnpublishVolume: called with args %+v", req)

	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(4).Infof("NodeUnpublishVolume: volumeId (%#v)", volID)

	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	_, refCount, err := driver.mounter.GetDeviceName(targetPath)
	if err != nil {
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
	// unmount bind mount
	err = driver.mounter.Unmount(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", targetPath, err)
	}
	klog.V(5).Infof("NodeUnpublishVolume: %s unmounted", targetPath)

	err = os.Remove(targetPath)
	if err != nil && !os.IsNotExist(err) {
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

	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target path not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	_, refCount, err := driver.mounter.GetDeviceName(targetPath)
	if err != nil {
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
	err = driver.mounter.Unmount(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", targetPath, err)
	}
	klog.V(5).Infof("NodeUnstageVolume: %s unmounted", targetPath)

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

func (driver *Driver) mountBind(sourcePath string, mntOptions []string, targetPath string) error {
	fsType := ""
	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)
	mountOptions = append(mountOptions, "bind")

	klog.V(5).Infof("Mounting %s at %s with options %v", sourcePath, targetPath, mountOptions)
	if err := driver.mounter.MountSensitive2(sourcePath, sourcePath, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", sourcePath, fsType, targetPath, err)
	}

	return nil
}

func (driver *Driver) mountFuse(volContext map[string]string, volSecrets map[string]string, mntOptions []string, targetPath string) error {
	enforceProxyAccess := driver.getDriverConfigEnforceProxyAccess()
	proxyUser := driver.getDriverConfigUser()

	irodsConn, err := ExtractIRODSConnectionInfo(volContext, volSecrets)
	if err != nil {
		return err
	}

	if enforceProxyAccess {
		if proxyUser == irodsConn.User {
			// same proxy user
			// enforce clientUser
			if len(irodsConn.ClientUser) == 0 {
				return status.Error(codes.InvalidArgument, "Argument clientUser must be given")
			}

			if irodsConn.User == irodsConn.ClientUser {
				return status.Errorf(codes.InvalidArgument, "Argument clientUser cannot be the same as user - user %s, clientUser %s", irodsConn.User, irodsConn.ClientUser)
			}
		} else {
			// replaced user
			// static volume provisioning takes user argument from pv
			// this is okay

			// do not allow anonymous access
			if irodsConn.User == "anonymous" {
				return status.Error(codes.InvalidArgument, "Argument user must be a non-anonymous user")
			}
		}
	}

	if irodsConn.ClientUser == "anonymous" {
		return status.Error(codes.InvalidArgument, "Argument clientUser must be a non-anonymous user")
	}

	volPath := ""
	if irodsConn.Path == "/" {
		volPath = irodsConn.Path
	} else {
		volPath = strings.TrimRight(irodsConn.Path, "/")
	}

	// need to check if mount path is in whitelist
	if !driver.isMountPathAllowed(volPath) {
		return status.Errorf(codes.InvalidArgument, "Argument volumeRootPath %s is not allowed to mount", volPath)
	}

	fsType := "irodsfs"
	source := "irodsfs" // device name -- this parameter is actually required but ignored

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	irodsFsConfig := &IRODSFSConfig{
		Host:       irodsConn.Hostname,
		Port:       irodsConn.Port,
		ProxyUser:  irodsConn.User,
		ClientUser: irodsConn.ClientUser,
		Zone:       irodsConn.Zone,
		Password:   irodsConn.Password,
		PathMappings: []IRODSFSPathMapping{
			{
				IRODSPath:    fmt.Sprintf("%s%s", irodsConn.Zone, volPath),
				MappingPath:  "/",
				ResourceType: "dir",
			},
		},
		AllowOther: true,

		CacheTimeout:     1 * time.Minute,
		CacheCleanupTime: 1 * time.Minute,
	}

	irodsFsConfigBytes, err := yaml.Marshal(irodsFsConfig)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not serialize configuration: %v", volPath, err)
	}

	mountOptions = append(mountOptions, mntOptions...)
	mountOptions = append(mountOptions, "config=-") // read configuration yaml via STDIN

	// passing configuration yaml via STDIN
	stdinArgs = append(stdinArgs, string(irodsFsConfigBytes))

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, targetPath, mountOptions)
	if err := driver.mounter.MountSensitive2(source, source, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", source, fsType, targetPath, err)
	}

	return nil
}

func (driver *Driver) mountWebdav(volContext map[string]string, volSecrets map[string]string, mntOptions []string, targetPath string) error {
	irodsConn, err := ExtractIRODSWebDAVConnectionInfo(volContext, volSecrets)
	if err != nil {
		return err
	}

	fsType := "davfs"
	source := irodsConn.URL

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	// if user == anonymous, password is empty, and doesn't need to pass user/password as arguments
	if len(irodsConn.User) > 0 && irodsConn.User != "anonymous" && len(irodsConn.Password) > 0 {
		mountSensitiveOptions = append(mountSensitiveOptions, fmt.Sprintf("username=%s", irodsConn.User))
		stdinArgs = append(stdinArgs, irodsConn.Password)
	}

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, targetPath, mountOptions)
	if err := driver.mounter.MountSensitive2(source, source, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", source, fsType, targetPath, err)
	}

	return nil
}

func (driver *Driver) mountNfs(volContext map[string]string, volSecrets map[string]string, mntOptions []string, targetPath string) error {
	irodsConn, err := ExtractIRODSNFSConnectionInfo(volContext, volSecrets)
	if err != nil {
		return err
	}

	fsType := "nfs"
	source := fmt.Sprintf("%s:%s", irodsConn.Hostname, irodsConn.Path)

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	if irodsConn.Port != 2049 {
		mountOptions = append(mountOptions, fmt.Sprintf("port=%d", irodsConn.Port))
	}

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, targetPath, mountOptions)
	if err := driver.mounter.MountSensitive2(source, source, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", source, fsType, targetPath, err)
	}

	return nil
}
