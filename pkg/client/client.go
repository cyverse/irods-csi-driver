package client

import (
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"

	"github.com/cyverse/irods-csi-driver/pkg/client/common"
	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"github.com/cyverse/irods-csi-driver/pkg/client/nfs"
	"github.com/cyverse/irods-csi-driver/pkg/client/webdav"
	"github.com/cyverse/irods-csi-driver/pkg/metrics"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
)

// MountClient mounts a fs client
func MountClient(mounter mounter.Mounter, volID string, configs map[string]string, mountOptions []string, targetPath string) error {
	irodsClientType := common.GetClientType(configs)
	switch irodsClientType {
	case common.IrodsFuseClientType:
		klog.V(5).Infof("mounting %q", irodsClientType)

		if err := irods.Mount(mounter, volID, configs, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
		return nil
	case common.WebdavClientType:
		klog.V(5).Infof("mounting %q", irodsClientType)

		if err := webdav.Mount(mounter, volID, configs, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
		return nil
	case common.NfsClientType:
		klog.V(5).Infof("mounting %q", irodsClientType)

		if err := nfs.Mount(mounter, volID, configs, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
		return nil
	default:
		metrics.IncreaseCounterForVolumeMountFailures()
		return status.Errorf(codes.Internal, "unknown driver type '%v'", irodsClientType)
	}
}

// UnmountClient unmounts a fs client
func UnmountClient(mounter mounter.Mounter, volID string, irodsClientType common.ClientType, configs map[string]string, targetPath string) error {
	switch irodsClientType {
	case common.IrodsFuseClientType:
		klog.V(5).Infof("unmounting %q", irodsClientType)

		if err := irods.Unmount(mounter, volID, configs, targetPath); err != nil {
			metrics.IncreaseCounterForVolumeUnmountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeUnmount()
		metrics.DecreaseCounterForActiveVolumeMount()
		return nil
	case common.WebdavClientType:
		klog.V(5).Infof("unmounting %q", irodsClientType)

		if err := webdav.Unmount(mounter, volID, configs, targetPath); err != nil {
			metrics.IncreaseCounterForVolumeUnmountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeUnmount()
		metrics.DecreaseCounterForActiveVolumeMount()
		return nil
	case common.NfsClientType:
		klog.V(5).Infof("unmounting %q", irodsClientType)

		if err := nfs.Unmount(mounter, volID, configs, targetPath); err != nil {
			metrics.IncreaseCounterForVolumeUnmountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeUnmount()
		metrics.DecreaseCounterForActiveVolumeMount()
		return nil
	default:
		metrics.IncreaseCounterForVolumeUnmountFailures()
		return status.Errorf(codes.Internal, "unknown driver type '%v'", irodsClientType)
	}
}
