package irods

import (
	"fmt"
	"os"
	"syscall"

	client_common "github.com/cyverse/irods-csi-driver/pkg/client/common"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"golang.org/x/xerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

func Mount(mounter mounter.Mounter, volID string, configs map[string]string, mntOptions []string, targetPath string) error {
	irodsConnectionInfo, err := GetConnectionInfo(configs)
	if err != nil {
		return err
	}

	// test connection creation to check account info is correct
	err = TestConnection(irodsConnectionInfo)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Could not create iRODS Conenction with given access parameters - %q", err)
	}

	fsType := "irodsfs"
	source := "irodsfs" // device name -- this parameter is actually required but ignored

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	irodsFsConfig := NewDefaultIRODSFSConfig()

	// create irodsfs dataroot
	dataRootPath := client_common.GetConfigDataRootPath(configs, volID)
	err = makeIrodsFuseLiteDataRootPath(dataRootPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	irodsFsConfig.DataRootPath = dataRootPath
	irodsFsConfig.Host = irodsConnectionInfo.Hostname
	irodsFsConfig.Port = irodsConnectionInfo.Port
	irodsFsConfig.ProxyUser = irodsConnectionInfo.User
	irodsFsConfig.ClientUser = irodsConnectionInfo.ClientUser
	irodsFsConfig.Zone = irodsConnectionInfo.Zone
	irodsFsConfig.Password = irodsConnectionInfo.Password
	irodsFsConfig.Resource = irodsConnectionInfo.Resource
	irodsFsConfig.MonitorURL = irodsConnectionInfo.MonitorURL
	irodsFsConfig.PathMappings = irodsConnectionInfo.PathMappings
	irodsFsConfig.NoPermissionCheck = irodsConnectionInfo.NoPermissionCheck
	irodsFsConfig.UID = irodsConnectionInfo.UID
	irodsFsConfig.GID = irodsConnectionInfo.GID
	irodsFsConfig.SystemUser = irodsConnectionInfo.SystemUser
	irodsFsConfig.PoolEndpoint = irodsConnectionInfo.PoolEndpoint
	irodsFsConfig.Profile = irodsConnectionInfo.Profile
	irodsFsConfig.ProfileServicePort = irodsConnectionInfo.ProfilePort
	irodsFsConfig.InstanceID = volID

	irodsFsConfigBytes, err := yaml.Marshal(irodsFsConfig)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not serialize configuration: %v", err)
	}

	// passing configuration yaml via STDIN
	stdinArgs = append(stdinArgs, string(irodsFsConfigBytes))

	irodsFSMountPath := targetPath

	// for overlayfs
	overlayFSLowerPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	err = makeOverlayFSPath(overlayFSLowerPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	overlayFSUpperPath := client_common.GetConfigOverlayFSUpperPath(configs, volID)
	err = makeOverlayFSPath(overlayFSUpperPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	overlayFSWorkDirPath := client_common.GetConfigOverlayFSWorkDirPath(configs, volID)
	err = makeOverlayFSPath(overlayFSWorkDirPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	overlayFSMountPath := targetPath

	if irodsConnectionInfo.OverlayFS {
		irodsFSMountPath = overlayFSLowerPath
	}

	// mount irodsfs
	mountOptions = append(mountOptions, mntOptions...)
	mountOptions = append(mountOptions, fmt.Sprintf("mounttimeout=%d", irodsConnectionInfo.MountTimeout))
	mountOptions = append(mountOptions, "config=-") // read configuration yaml via STDIN

	klog.V(5).Infof("Mounting %q (%q) at %q with options %v", source, fsType, irodsFSMountPath, mountOptions)
	if err := mounter.MountSensitive2(source, source, irodsFSMountPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		// umount the volume to ensure no leftovers
		mounter.UnmountLazy(irodsFSMountPath, true)

		return status.Errorf(codes.Internal, "Failed to mount %q (%q) at %q: %v", source, fsType, irodsFSMountPath, err)
	}

	// mount overlayfs
	if irodsConnectionInfo.OverlayFS {
		overlayfsMountOptions := []string{}
		overlayfsMountOptions = append(overlayfsMountOptions, fmt.Sprintf("lowerdir=%q,upperdir=%q,workdir=%q", overlayFSLowerPath, overlayFSUpperPath, overlayFSWorkDirPath))

		overlayfsMountSensitiveOptions := []string{}

		klog.V(5).Infof("Mounting %q (%q) at %q with options %v", "overlay", "overlay", overlayFSMountPath, overlayfsMountOptions)
		if err := mounter.MountSensitive("overlay", overlayFSMountPath, "overlay", overlayfsMountOptions, overlayfsMountSensitiveOptions); err != nil {
			// umount the volume to ensure no leftovers
			mounter.Unmount(overlayFSMountPath)

			return status.Errorf(codes.Internal, "Failed to mount %q (%q) at %q: %v", "overlay", "overlay", overlayFSMountPath, err)
		}
	}

	return nil
}

func Unmount(mounter mounter.Mounter, volID string, configs map[string]string, targetPath string) error {
	irodsConnectionInfo, err := GetConnectionInfo(configs)
	if err != nil {
		return err
	}

	irodsFSMountPath := targetPath
	dataRootPath := client_common.GetConfigDataRootPath(configs, volID)

	// for overlayfs
	overlayFSLowerPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	overlayFSUpperPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	overlayFSWorkDirPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	overlayFSMountPath := targetPath

	if irodsConnectionInfo.OverlayFS {
		// overlayfs
		irodsFSMountPath = overlayFSLowerPath

		klog.V(5).Infof("Unmounting %q (%q) at %q", "overlay", "overlay", overlayFSMountPath)

		err = mounter.UnmountLazy(overlayFSMountPath, true)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to unmount %q: %v", overlayFSMountPath, err)
		}

		// delete workdir
		err = deleteOverlayFSData(overlayFSWorkDirPath)
		if err != nil {
			klog.V(3).Infof("Error deleting overlayfs workdir data at %q - ignoring", overlayFSWorkDirPath)
		}

		klog.V(5).Infof("Unmounting %q (%q) at %q", "irodsfs", "irodsfs", irodsFSMountPath)

		// irodsfs
		err = mounter.UnmountLazy(irodsFSMountPath, true)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to unmount %q: %v", irodsFSMountPath, err)
		}

		err = deleteIrodsFuseLiteData(dataRootPath)
		if err != nil {
			klog.V(3).Infof("Error deleting iRODS FUSE Lite data at %q - ignoring", dataRootPath)
		}

		// sync
		// this takes some time if there were a lot of file changes
		// so we will do this asynchronously
		go func() {
			klog.V(5).Infof("Synching overlayfs at %q", overlayFSMountPath)

			err = syncOverlayFS(irodsConnectionInfo, overlayFSUpperPath)
			if err != nil {
				klog.V(3).Infof("Error syncing overlayfs upper data at %q - ignoring", overlayFSUpperPath)
			}

			klog.V(5).Infof("Done synching overlayfs at %q", overlayFSMountPath)

			// delete upper
			err = deleteOverlayFSData(overlayFSUpperPath)
			if err != nil {
				klog.V(3).Infof("Error deleting overlayfs upper data at %q - ignoring", overlayFSUpperPath)
			}
		}()
	} else {
		// it is safe to unmount now
		klog.V(5).Infof("Unmounting %q (%q) at %q", "irodsfs", "irodsfs", irodsFSMountPath)

		err = mounter.UnmountLazy(irodsFSMountPath, true)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to unmount %q: %v", irodsFSMountPath, err)
		}

		err = deleteIrodsFuseLiteData(dataRootPath)
		if err != nil {
			klog.V(3).Infof("Error deleting iRODS FUSE Lite data at %q - ignoring", dataRootPath)
		}
	}

	return nil
}

func makeIrodsFuseLiteDataRootPath(dataRootPath string) error {
	// create irodsfs dataroot
	_, err := os.Stat(dataRootPath)
	if err != nil {
		if os.IsNotExist(err) {
			// not exist, make one
			oldMask := syscall.Umask(0)
			defer syscall.Umask(oldMask)

			err = os.MkdirAll(dataRootPath, os.FileMode(0777))
			if err != nil {
				return xerrors.Errorf("failed to create a irodsfs data root path %q: %w", dataRootPath, err)
			}
		} else {
			return xerrors.Errorf("failed to access a irodsfs data root path %q: %w", dataRootPath, err)
		}
	}
	return nil
}

func makeOverlayFSPath(overlayfsPath string) error {
	// create
	_, err := os.Stat(overlayfsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// not exist, make one
			oldMask := syscall.Umask(0)
			defer syscall.Umask(oldMask)

			err = os.MkdirAll(overlayfsPath, os.FileMode(0777))
			if err != nil {
				return xerrors.Errorf("failed to create a overlayfs data path %q: %w", overlayfsPath, err)
			}
		} else {
			return xerrors.Errorf("failed to access a overlayfs data path %q: %w", overlayfsPath, err)
		}
	}
	return nil
}

func deleteIrodsFuseLiteData(dataRootPath string) error {
	// delete irodsfs data
	err := os.RemoveAll(dataRootPath)
	if err != nil {
		return xerrors.Errorf("failed to delete a irodsfs data root path %q: %w", dataRootPath, err)
	}
	return nil
}

func syncOverlayFS(connectionInfo *IRODSFSConnectionInfo, upperPath string) error {
	syncher, err := NewOverlayFSSyncher(connectionInfo, upperPath)
	if err != nil {
		return xerrors.Errorf("failed to create a overlayfs syncher: %w", err)
	}

	err = syncher.Sync()
	if err != nil {
		return xerrors.Errorf("failed to sync overlayfs upper %q: %w", err, upperPath)
	}

	syncher.Release()

	return nil
}

func deleteOverlayFSData(dataPath string) error {
	// delete upper
	err := os.RemoveAll(dataPath)
	if err != nil {
		return xerrors.Errorf("failed to delete a overlayfs data path %q: %w", dataPath, err)
	}
	return nil
}
