package client

import (
	"os"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"

	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"github.com/cyverse/irods-csi-driver/pkg/client/nfs"
	"github.com/cyverse/irods-csi-driver/pkg/client/webdav"
	"github.com/cyverse/irods-csi-driver/pkg/metrics"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
)

// ClientType is a mount client type
type ClientType string

// mount driver (iRODS Client) types
const (
	// IrodsFuseClientType is for iRODS FUSE
	IrodsFuseClientType ClientType = "irodsfuse"
	// WebdavClientType is for WebDav client (Davfs2)
	WebdavClientType ClientType = "webdav"
	// NfsClientType is for NFS client
	NfsClientType ClientType = "nfs"
)

// GetClientType returns iRODS Client value from param map
func GetClientType(params map[string]string) ClientType {
	irodsClient := ""
	for k, v := range params {
		if strings.ToLower(k) == "client" {
			irodsClient = v
			break
		}
	}

	return GetValidClientType(irodsClient)
}

// IsValidClientType checks if given client string is valid
func IsValidClientType(client string) bool {
	switch client {
	case string(IrodsFuseClientType):
		return true
	case string(WebdavClientType):
		return true
	case string(NfsClientType):
		return true
	default:
		return false
	}
}

// GetValidClientType checks if given client string is valid
func GetValidClientType(client string) ClientType {
	switch client {
	case string(IrodsFuseClientType):
		return IrodsFuseClientType
	case string(WebdavClientType):
		return WebdavClientType
	case string(NfsClientType):
		return NfsClientType
	default:
		return IrodsFuseClientType
	}
}

func MountClient(mounter mounter.Mounter, configs map[string]string, mountOptions []string, targetPath string) error {
	irodsClientType := GetClientType(configs)
	switch irodsClientType {
	case IrodsFuseClientType:
		klog.V(5).Infof("mounting %s", irodsClientType)

		if err := irods.Mount(mounter, configs, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
		return nil
	case WebdavClientType:
		klog.V(5).Infof("mounting %s", irodsClientType)

		if err := webdav.Mount(mounter, configs, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
		return nil
	case NfsClientType:
		klog.V(5).Infof("mounting %s", irodsClientType)

		if err := nfs.Mount(mounter, configs, mountOptions, targetPath); err != nil {
			os.Remove(targetPath)
			metrics.IncreaseCounterForVolumeMountFailures()
			return err
		}

		metrics.IncreaseCounterForVolumeMount()
		metrics.IncreaseCounterForActiveVolumeMount()
		return nil
	default:
		metrics.IncreaseCounterForVolumeMountFailures()
		return status.Errorf(codes.Internal, "unknown driver type - %v", irodsClientType)
	}

	return nil
}
