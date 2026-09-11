package driver

import (
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"k8s.io/utils/mount"
)

// Mounter is responsible only for local CSI bind mounts. Filesystem mounts are
// created and removed by irodsfsd.
type Mounter interface {
	mount.Interface
}

// NewNodeMounter returns the standard Kubernetes mounter for local bind mount
// operations in the node service.
func NewNodeMounter() Mounter {
	return mount.New("")
}

func MountBind(mounter Mounter, sourcePath string, mntOptions []string, targetPath string) error {
	mountOptions := append([]string{}, mntOptions...)
	mountOptions = append(mountOptions, "bind")

	klog.V(5).Infof("Mounting %q at %q with options %v", sourcePath, targetPath, mountOptions)
	if err := mounter.Mount(sourcePath, targetPath, "", mountOptions); err != nil {
		return status.Errorf(codes.Internal, "could not bind mount %q at %q: %v", sourcePath, targetPath, err)
	}

	return nil
}

// PathExists returns true if path exists.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// MakeDir creates path and its missing parents.
func MakeDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// GetMountOptions returns mount options from VolumeCapability_MountVolume.
func GetMountOptions(volume *csi.VolumeCapability_MountVolume, accessMode *csi.VolumeCapability_AccessMode) []string {
	mountOptions := []string{}
	if volume != nil {
		for _, option := range volume.MountFlags {
			if !hasValue(mountOptions, option) {
				mountOptions = append(mountOptions, option)
			}
		}
	}

	if accessMode != nil {
		switch accessMode.GetMode() {
		case csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
			csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY:
			if !hasValue(mountOptions, "ro") {
				mountOptions = append(mountOptions, "ro")
			}
		}
	}

	return mountOptions
}

func hasValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
