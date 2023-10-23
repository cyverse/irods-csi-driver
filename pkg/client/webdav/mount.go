package webdav

import (
	"fmt"
	"os"
	"path/filepath"

	client_common "github.com/cyverse/irods-csi-driver/pkg/client/common"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"golang.org/x/xerrors"
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

	// create davfs dataroot
	dataRootPath := client_common.GetConfigDataRootPath(configs, volID)
	err = makeDavFSDataRootPath(dataRootPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	err = makeDavFSCachePath(dataRootPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	davFsConfig := NewDefaultDavFSConfig()
	davFsConfig.AddParams(irodsConnectionInfo.Config)

	cachePath := getDavFSCachePath(dataRootPath)
	davFsConfig.AddParam("cache_dir", cachePath)

	configPath := filepath.Join(dataRootPath, "davfs2.conf")
	err = davFsConfig.SaveToFile(configPath)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	configOption := fmt.Sprintf("conf=%s", configPath)
	mountOptions = append(mountOptions, configOption)

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
	err := mounter.FuseUnmount(targetPath, true)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to unmount %q: %v", targetPath, err)
	}

	dataRootPath := client_common.GetConfigDataRootPath(configs, volID)
	err = deleteDavFSData(dataRootPath)
	if err != nil {
		klog.V(5).Infof("Error deleting davfs data at %s - ignoring", dataRootPath)
	}
	return nil
}

func makeDavFSDataRootPath(dataRootPath string) error {
	// create davfs dataroot
	_, err := os.Stat(dataRootPath)
	if err != nil {
		if os.IsNotExist(err) {
			// not exist, make one
			err = os.MkdirAll(dataRootPath, os.FileMode(0777))
			if err != nil {
				return xerrors.Errorf("failed to create a davfs data root path %s: %w", dataRootPath, err)
			}
		} else {
			return xerrors.Errorf("failed to access a davfs data root path %s: %w", dataRootPath, err)
		}
	}
	return nil
}

func getDavFSCachePath(dataRootPath string) string {
	return filepath.Join(dataRootPath, "cache")
}

func makeDavFSCachePath(dataRootPath string) error {
	// create davfs cache
	cachePath := getDavFSCachePath(dataRootPath)
	_, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// not exist, make one
			// cache dir must have permission 0777
			err = os.MkdirAll(cachePath, os.FileMode(0777))
			if err != nil {
				return xerrors.Errorf("failed to create a davfs cache path %s: %w", cachePath, err)
			}
		} else {
			return xerrors.Errorf("failed to access a davfs cache path %s: %w", cachePath, err)
		}
	}
	return nil
}

func deleteDavFSData(dataRootPath string) error {
	// delete davfs data
	err := os.RemoveAll(dataRootPath)
	if err != nil {
		return xerrors.Errorf("failed to delete a davfs data root path %s: %w", dataRootPath, err)
	}
	return nil
}
