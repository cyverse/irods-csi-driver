package mounter

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func MountBind(mounter Mounter, sourcePath string, mntOptions []string, targetPath string) error {
	fsType := ""
	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)
	mountOptions = append(mountOptions, "bind")

	klog.V(5).Infof("Mounting %q at %q with options %v", sourcePath, targetPath, mountOptions)
	if err := mounter.MountSensitive2(sourcePath, sourcePath, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Could not mount %q (%q) at %q: %v", sourcePath, fsType, targetPath, err)
	}

	return nil
}
