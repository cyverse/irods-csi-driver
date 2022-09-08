package nfs

import (
	"fmt"

	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

func Mount(mounter mounter.Mounter, configs map[string]string, mntOptions []string, targetPath string) error {
	irodsConnectionInfo, err := GetConnectionInfo(configs)
	if err != nil {
		return err
	}

	fsType := "nfs"
	source := fmt.Sprintf("%s:%s", irodsConnectionInfo.Hostname, irodsConnectionInfo.Path)

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	mountOptions = append(mountOptions, mntOptions...)

	if irodsConnectionInfo.Port != 2049 {
		mountOptions = append(mountOptions, fmt.Sprintf("port=%d", irodsConnectionInfo.Port))
	}

	klog.V(5).Infof("Mounting %s (%s) at %s with options %v", source, fsType, targetPath, mountOptions)
	if err := mounter.MountSensitive2(source, source, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		return status.Errorf(codes.Internal, "Failed to mount %q (%q) at %q: %v", source, fsType, targetPath, err)
	}

	return nil
}
