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
	klog.V(5).Infof("Testing iRODS connection")
	err = TestConnection(irodsConnectionInfo)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Could not create iRODS Conenction with given access parameters - %v", err)
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

	if !irodsConnectionInfo.OverlayFS {
		// mount irodsfs
		mountOptions = append(mountOptions, mntOptions...)
		mountOptions = append(mountOptions, fmt.Sprintf("mounttimeout=%d", irodsConnectionInfo.MountTimeout))
		mountOptions = append(mountOptions, "config=-") // read configuration yaml via STDIN

		klog.V(5).Infof("Mounting %q (%q) at %q with options %v", source, fsType, targetPath, mountOptions)
		if err := mounter.MountSensitive2(source, source, targetPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
			// umount the volume to ensure no leftovers
			mounter.UnmountLazy(targetPath, true)

			return status.Errorf(codes.Internal, "Failed to mount %q (%q) at %q: %v", source, fsType, targetPath, err)
		}
		return nil
	}

	// mount irodsfs and overlayfs
	if !IsOverlayDriverSupported() {
		// kernel does not support overlay driver
		// use fuse-overlayfs
		klog.V(5).Infof("Host kernel does not support stable overlay fs, going to use fuse-overlayfs")
		irodsConnectionInfo.OverlayFSDriver = FuseOverlayFSDriverType
	}

	overlayFSLowerPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	err = makeOverlayFSPath(overlayFSLowerPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	// mount irodsfs
	mountOptions = append(mountOptions, mntOptions...)
	mountOptions = append(mountOptions, fmt.Sprintf("mounttimeout=%d", irodsConnectionInfo.MountTimeout))
	mountOptions = append(mountOptions, "config=-") // read configuration yaml via STDIN

	klog.V(5).Infof("Mounting %q (%q) at %q with options %v", source, fsType, overlayFSLowerPath, mountOptions)
	if err := mounter.MountSensitive2(source, source, overlayFSLowerPath, fsType, mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		// umount the volume to ensure no leftovers
		mounter.UnmountLazy(overlayFSLowerPath, true)

		return status.Errorf(codes.Internal, "Failed to mount %q (%q) at %q: %v", source, fsType, overlayFSLowerPath, err)
	}

	if irodsConnectionInfo.OverlayFSDriver == FuseOverlayFSDriverType {
		err = mountFuseOverlayFS(mounter, irodsConnectionInfo, volID, configs, targetPath)
		if err != nil {
			return err
		}
	} else {
		err = mountOverlay(mounter, irodsConnectionInfo, volID, configs, targetPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func mountOverlay(mounter mounter.Mounter, irodsConnectionInfo *IRODSFSConnectionInfo, volID string, configs map[string]string, mountPath string) error {
	lowerPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	upperPath := client_common.GetConfigOverlayFSUpperPath(configs, volID)
	workdirPath := client_common.GetConfigOverlayFSWorkDirPath(configs, volID)

	// lower is already created
	err := makeOverlayFSPath(upperPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	err = makeOverlayFSPath(workdirPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	mountOptions := []string{}
	mountOptions = append(mountOptions, fmt.Sprintf("lowerdir=%s", lowerPath))
	mountOptions = append(mountOptions, fmt.Sprintf("upperdir=%s", upperPath))
	mountOptions = append(mountOptions, fmt.Sprintf("workdir=%s", workdirPath))
	mountOptions = append(mountOptions, "xino=off")

	mountSensitiveOptions := []string{}

	klog.V(5).Infof("Mounting overlay at %q with options %v", mountPath, mountOptions)
	if err := mounter.MountSensitive("overlay", mountPath, "overlay", mountOptions, mountSensitiveOptions); err != nil {
		// umount the volume to ensure no leftovers
		mounter.Unmount(mountPath)

		return status.Errorf(codes.Internal, "Failed to mount overlay at %q: %v", mountPath, err)
	}

	return nil
}

func mountFuseOverlayFS(mounter mounter.Mounter, irodsConnectionInfo *IRODSFSConnectionInfo, volID string, configs map[string]string, mountPath string) error {
	lowerPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	upperPath := client_common.GetConfigOverlayFSUpperPath(configs, volID)
	workdirPath := client_common.GetConfigOverlayFSWorkDirPath(configs, volID)

	err := makeOverlayFSPath(lowerPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	err = makeOverlayFSPath(upperPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	err = makeOverlayFSPath(workdirPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	mountOptions := []string{}
	mountOptions = append(mountOptions, fmt.Sprintf("lowerdir=%s", lowerPath))
	mountOptions = append(mountOptions, fmt.Sprintf("upperdir=%s", upperPath))
	mountOptions = append(mountOptions, fmt.Sprintf("workdir=%s", workdirPath))
	mountOptions = append(mountOptions, fmt.Sprintf("squash_to_uid=%d", irodsConnectionInfo.UID))
	mountOptions = append(mountOptions, fmt.Sprintf("squash_to_gid=%d", irodsConnectionInfo.GID))
	mountOptions = append(mountOptions, "static_nlink")
	mountOptions = append(mountOptions, "noacl")

	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	klog.V(5).Infof("Mounting fuse-overlayfs at %q with options %v", mountPath, mountOptions)
	if err := mounter.MountSensitive2("fuseoverlayfs", "fuseoverlayfs", mountPath, "fuseoverlayfs", mountOptions, mountSensitiveOptions, stdinArgs); err != nil {
		// umount the volume to ensure no leftovers
		mounter.UnmountLazy(mountPath, true)

		return status.Errorf(codes.Internal, "Failed to mount fuse-overlayfs at %q: %v", mountPath, err)
	}

	return nil
}

func Unmount(mounter mounter.Mounter, volID string, configs map[string]string, targetPath string) error {
	irodsConnectionInfo, err := GetConnectionInfo(configs)
	if err != nil {
		return err
	}

	dataRootPath := client_common.GetConfigDataRootPath(configs, volID)

	if !irodsConnectionInfo.OverlayFS {
		// unmount irodsfs
		klog.V(5).Infof("Unmounting irodsfs at %q", targetPath)

		err = mounter.UnmountLazy(targetPath, true)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to unmount %q: %v", targetPath, err)
		}

		err = deleteIrodsFuseLiteData(dataRootPath)
		if err != nil {
			klog.Errorf("Error deleting iRODS FUSE Lite data at %q, %s, ignoring", dataRootPath, err)
		}
		return nil
	}

	// unmount irodsfs and overlayfs
	err = unmountOverlayFS(mounter, irodsConnectionInfo, volID, configs, targetPath)
	if err != nil {
		return err
	}

	lowerPath := client_common.GetConfigOverlayFSLowerPath(configs, volID)
	upperPath := client_common.GetConfigOverlayFSUpperPath(configs, volID)

	klog.V(5).Infof("Unmounting irodsfs at %q", lowerPath)

	err = mounter.UnmountLazy(lowerPath, true)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to unmount %q: %v", lowerPath, err)
	}

	err = deleteIrodsFuseLiteData(dataRootPath)
	if err != nil {
		klog.Errorf("Error deleting iRODS FUSE Lite data at %q, %s, ignoring", dataRootPath, err)
	}

	err = deleteOverlayFSData(lowerPath)
	if err != nil {
		klog.Errorf("Error deleting overlayfs lower data at %q, %s, ignoring", lowerPath, err)
	}

	// sync
	// this takes some time if there were a lot of file changes
	// so we will do this asynchronously
	go func() {
		klog.V(5).Infof("Synching overlayfs upper data at %q", upperPath)

		err = syncOverlayFS(irodsConnectionInfo, upperPath)
		if err != nil {
			klog.Errorf("Error syncing overlayfs upper data at %q, %s, ignoring", upperPath, err)
		}

		klog.V(5).Infof("Done synching overlayfs upper data at %q", upperPath)

		// delete upper
		err = deleteOverlayFSData(upperPath)
		if err != nil {
			klog.Errorf("Error deleting overlayfs upper data at %q, %s, ignoring", upperPath, err)
		}
	}()

	return nil
}

func unmountOverlayFS(mounter mounter.Mounter, irodsConnectionInfo *IRODSFSConnectionInfo, volID string, configs map[string]string, mountPath string) error {
	workdirPath := client_common.GetConfigOverlayFSWorkDirPath(configs, volID)

	// overlayfs
	klog.V(5).Infof("Unmounting overlayfs at %q", mountPath)

	err := mounter.UnmountLazy(mountPath, true)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to unmount %q: %v", mountPath, err)
	}

	// delete workdir
	err = deleteOverlayFSData(workdirPath)
	if err != nil {
		klog.Errorf("Error deleting overlayfs workdir data at %q, %s, ignoring", workdirPath, err)
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
		return xerrors.Errorf("failed to sync overlayfs upper %q: %w", upperPath, err)
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
