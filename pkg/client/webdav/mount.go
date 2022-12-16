package webdav

import (
	"fmt"

	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

func Mount(mounter mounter.Mounter, volID string, configs map[string]string, mntOptions []string, targetPath string) error {
	irodsConnectionInfo, err := GetConnectionInfo(configs)
	if err != nil {
		return err
	}

	fsType := "davfs"
	source := irodsConnectionInfo.URL

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	// if user == anonymous, password is empty, and doesn't need to pass user/password as arguments
	if len(irodsConnectionInfo.User) > 0 && !irodsConnectionInfo.IsAnonymousUser() && len(irodsConnectionInfo.Password) > 0 {
		mountSensitiveOptions = append(mountSensitiveOptions, fmt.Sprintf("username=%s", irodsConnectionInfo.User))
		stdinArgs = append(stdinArgs, irodsConnectionInfo.Password)
	}

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, targetPath, mountOptions)
	if err := mounter.MountSensitive2(source, source, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Failed to mount %q (%q) at %q: %v", source, fsType, targetPath, err)
	}

	return nil
}

func Unmount(mounter mounter.Mounter, volID string, configs map[string]string, targetPath string) error {
	err := mounter.UnmountForcefully(targetPath)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to unmount %q: %v", targetPath, err)
	}
	return nil
}
