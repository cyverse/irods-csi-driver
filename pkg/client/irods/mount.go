package irods

import (
	"fmt"
	"os"

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
		return status.Errorf(codes.InvalidArgument, "Could not create iRODS Conenction with given access parameters - %s", err.Error())
	}

	fsType := "irodsfs"
	source := "irodsfs" // device name -- this parameter is actually required but ignored

	mountOptions := []string{}
	mountSensitiveOptions := []string{}
	stdinArgs := []string{}

	irodsFsConfig := NewDefaultIRODSFSConfig()

	// create irodsfs dataroot
	dataRootPath := getConfigDataRootPath(configs, volID)
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

	mountOptions = append(mountOptions, mntOptions...)
	mountOptions = append(mountOptions, fmt.Sprintf("mounttimeout=%d", irodsConnectionInfo.MountTimeout))
	mountOptions = append(mountOptions, "config=-") // read configuration yaml via STDIN

	// passing configuration yaml via STDIN
	stdinArgs = append(stdinArgs, string(irodsFsConfigBytes))

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

	// manage logs
	dataRootPath := getConfigDataRootPath(configs, volID)
	err = deleteIrodsFuseLiteData(dataRootPath)
	if err != nil {
		klog.V(5).Infof("Error deleting iRODS FUSE Lite data at %s - ignoring", dataRootPath)
	}

	return nil
}

func makeIrodsFuseLiteDataRootPath(dataRootPath string) error {
	// create irodsfs dataroot
	_, err := os.Stat(dataRootPath)
	if err != nil {
		if os.IsNotExist(err) {
			// not exist, make one
			err = os.MkdirAll(dataRootPath, os.FileMode(0755))
			if err != nil {
				return xerrors.Errorf("failed to create a irodsfs data root path %s: %w", dataRootPath, err)
			}
		} else {
			return xerrors.Errorf("failed to access a irodsfs data root path %s: %w", dataRootPath, err)
		}
	}
	return nil
}

func deleteIrodsFuseLiteData(dataRootPath string) error {
	// delete irodsfs data
	err := os.RemoveAll(dataRootPath)
	if err != nil {
		return xerrors.Errorf("failed to delete a irodsfs data root path %s: %w", dataRootPath, err)
	}
	return nil
}
