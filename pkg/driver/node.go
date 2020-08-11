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
	"os"
	"path/filepath"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var (
	nodeCaps             = []csi.NodeServiceCapability_RPC_Type{}
	volumeCapAccessModes = []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}
)

const (
	FuseType   = "irodsfuse"
	WebdavType = "webdav"
	NfsType    = "nfs"

	sensitiveArgsRemoved = "<masked>"
)

// this is for dynamic volume provisioning
func (driver *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// this is for dynamic volume provisioning
func (driver *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (driver *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.V(4).Infof("NodePublishVolume: called with args %+v", req)

	target := req.GetTargetPath()
	if len(target) == 0 {
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

	// this is volumeHandle -- we don't use this
	//volumeId := req.GetVolumeId()
	irodsClient := FuseType
	volContext := req.GetVolumeContext()
	for k, v := range volContext {
		if strings.ToLower(k) == "driver" || strings.ToLower(k) == "client" {
			irodsClient = v
		}
	}

	klog.V(5).Infof("NodePublishVolume: creating dir %s", target)
	if err := MakeDir(target); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", target, err)
	}

	switch irodsClient {
	case FuseType:
		klog.V(5).Infof("NodePublishVolume: mounting %s", irodsClient)
		if err := driver.mountFuse(volContext, mountOptions, target); err != nil {
			os.Remove(target)
			return nil, err
		}
	case WebdavType:
		klog.V(5).Infof("NodePublishVolume: mounting %s", irodsClient)
		if err := driver.mountWebdav(volContext, mountOptions, target); err != nil {
			os.Remove(target)
			return nil, err
		}
	case NfsType:
		klog.V(5).Infof("NodePublishVolume: mounting %s", irodsClient)
		if err := driver.mountNfs(volContext, mountOptions, target); err != nil {
			os.Remove(target)
			return nil, err
		}
	default:
		os.Remove(target)
		return nil, status.Errorf(codes.Internal, "unknown driver type - %v", irodsClient)
	}

	klog.V(5).Infof("NodePublishVolume: %s was mounted", target)
	return &csi.NodePublishVolumeResponse{}, nil
}

func (driver *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(4).Infof("NodeUnpublishVolume: called with args %+v", req)

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	_, refCount, err := driver.mounter.GetDeviceName(target)
	if err != nil {
		msg := fmt.Sprintf("failed to check if volume is mounted: %v", err)
		return nil, status.Error(codes.Internal, msg)
	}

	// From the spec: If the volume corresponding to the volume_id
	// is not staged to the staging_target_path, the Plugin MUST
	// reply 0 OK.
	if refCount == 0 {
		klog.V(5).Infof("NodeUnpublishVolume: %s target not mounted", target)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	klog.V(5).Infof("NodeUnpublishVolume: unmounting %s", target)
	err = driver.mounter.Unmount(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", target, err)
	}
	klog.V(5).Infof("NodeUnpublishVolume: %s unmounted", target)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (driver *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (driver *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

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

func (driver *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.V(4).Infof("NodeGetInfo: called with args %+v", req)

	return &csi.NodeGetInfoResponse{
		NodeId: driver.config.NodeID,
	}, nil
}

func (driver *Driver) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, m := range volumeCapAccessModes {
			if m == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
		}
	}
	return foundAll
}

func (driver *Driver) mountFuse(volContext map[string]string, mntOptions []string, target string) error {
	var user, password, host, zone, ticket string

	port := 1247
	path := "/"

	for k, v := range volContext {
		switch strings.ToLower(k) {
		case "driver":
			// do nothing
			continue
		case "client":
			// do nothing - same as driver
			continue
		case "user":
			user = v
		case "password":
			password = v
		case "host":
			host = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be a valid port number - %s", k, err))
			}
			port = p
		case "ticket":
			// ticket is optional
			ticket = v
		case "zone":
			zone = v
		case "path":
			if !filepath.IsAbs(v) {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be an absolute path", k))
			}
			path = v
		default:
			return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %s not supported", k))
		}
	}

	if len(user) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("user specified (%s) is invalid", user))
	}

	if len(password) == 0 {
		return status.Error(codes.InvalidArgument, "user password specified is invalid")
	}

	if len(host) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("host specified (%s) is invalid", host))
	}

	if len(zone) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("zone specified (%s) is invalid", zone))
	}

	fsType := "irodsfs"
	source := fmt.Sprintf("irods://%s@%s:%d/%s/%s", user, host, port, zone, path[1:])

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	if len(ticket) > 0 {
		mountSensitiveOptions = append(mountSensitiveOptions, fmt.Sprintf("ticket=%s", ticket))
	}

	stdinArgs = append(stdinArgs, password)

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, target, mountOptions)
	if err := driver.mounter.MountSensitive2(source, source, target, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", source, fsType, target, err)
	}

	return nil
}

func (driver *Driver) mountWebdav(volContext map[string]string, mntOptions []string, target string) error {
	var user, password, host, zone, urlprefix string

	protocol := "https"
	port := 0
	path := "/"

	for k, v := range volContext {
		switch strings.ToLower(k) {
		case "driver":
			// do nothing
			continue
		case "client":
			// do nothing - same as driver
			continue
		case "protocol":
			switch strings.ToLower(v) {
			case "http":
			case "https":
				protocol = v
			default:
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be a valid protocol scheme - %s", k, v))
			}
		case "user":
			user = v
		case "password":
			password = v
		case "host":
			host = v
		case "urlprefix":
			urlprefix = v
		case "rootdir":
			urlprefix = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be a valid port number - %s", k, err))
			}
			port = p
		case "zone":
			zone = v
		case "path":
			if !filepath.IsAbs(v) {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be an absolute path", k))
			}
			path = v
		case "url":
			u, err := url.Parse(v)
			if err != nil {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be a valid url - %s", k, err))
			}

			protocol = strings.ToLower(u.Scheme)
			if u.User != nil {
				u_user := u.User.Username()
				if len(u_user) > 0 {
					user = u_user
				}

				u_password, u_set := u.User.Password()
				if u_set && len(u_password) > 0 {
					password = u_password
				}
			}

			u_host, u_port, _ := net.SplitHostPort(u.Host)
			host = u_host

			p, err := strconv.Atoi(u_port)
			if err == nil {
				port = p
			}

			path = u.Path
			zone = ""
			urlprefix = ""
		default:
			return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %s not supported", k))
		}
	}

	if len(host) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("host specified (%s) is invalid", host))
	}

	fsType := "davfs"
	if len(urlprefix) > 0 {
		urlprefix = strings.Trim(urlprefix, "/")
		urlprefix += "/"
	}

	var source string
	if len(zone) == 0 && len(urlprefix) == 0 {
		// via "url"
		if port > 0 {
			source = fmt.Sprintf("%s://%s:%d/%s", protocol, host, port, path[1:])
		} else {
			source = fmt.Sprintf("%s://%s/%s", protocol, host, path[1:])
		}
	} else {
		if port > 0 {
			source = fmt.Sprintf("%s://%s:%d/%s%s/%s", protocol, host, port, urlprefix, zone, path[1:])
		} else {
			source = fmt.Sprintf("%s://%s/%s%s/%s", protocol, host, urlprefix, zone, path[1:])
		}
	}

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	if len(user) > 0 && len(password) > 0 {
		mountSensitiveOptions = append(mountSensitiveOptions, fmt.Sprintf("username=%s", user))
		stdinArgs = append(stdinArgs, password)
	}

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, target, mountOptions)
	if err := driver.mounter.MountSensitive2(source, source, target, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", source, fsType, target, err)
	}

	return nil
}

func (driver *Driver) mountNfs(volContext map[string]string, mntOptions []string, target string) error {
	var host string

	port := 2049
	path := "/"

	for k, v := range volContext {
		switch strings.ToLower(k) {
		case "driver":
			// do nothing
			continue
		case "client":
			// do nothing - same as driver
			continue
		case "host":
			host = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be a valid port number - %s", k, err))
			}
			port = p
		case "path":
			if !filepath.IsAbs(v) {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %q must be an absolute path", k))
			}
			path = v
		default:
			return status.Error(codes.InvalidArgument, fmt.Sprintf("Volume context property %s not supported", k))
		}
	}

	if len(host) == 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("host specified (%s) is invalid", host))
	}

	fsType := "nfs"
	source := fmt.Sprintf("%s:%s", host, path)

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	if port != 2049 {
		mountOptions = append(mountOptions, fmt.Sprintf("port=%d", port))
	}

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, target, mountOptions)
	if err := driver.mounter.MountSensitive2(source, source, target, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", source, fsType, target, err)
	}

	return nil
}
