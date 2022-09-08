package irods

import (
	"fmt"

	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

func Mount(mounter mounter.Mounter, configs map[string]string, mntOptions []string, targetPath string) error {
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
