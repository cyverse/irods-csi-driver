package client

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"

	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"github.com/cyverse/irods-csi-driver/pkg/client/nfs"
	"github.com/cyverse/irods-csi-driver/pkg/client/webdav"
	"github.com/cyverse/irods-csi-driver/pkg/commons"
	irodsfsd_client "github.com/cyverse/irodsfsd/client"
)

// ValidateConfig verifies that configs can be used by the selected filesystem
// client without creating a mount.
func ValidateConfig(configs map[string]string) error {
	clientType, err := commons.ParseClientType(configs)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	switch clientType {
	case commons.IrodsFuseClientType:
		_, err = irods.GetConnectionInfo(configs)
	case commons.WebdavClientType:
		_, err = webdav.GetConnectionInfo(configs)
	case commons.NfsClientType:
		_, err = nfs.GetConnectionInfo(configs)
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported client type %q", clientType)
	}
	return err
}

// MountClient mounts the selected filesystem client. The caller owns target
// directory cleanup; this function owns only the mount lifecycle.
func MountClient(irodsfsdClient *irodsfsd_client.MountServiceClient, volID string, configs map[string]string, mountOptions []string, targetPath string) error {
	clientType, err := commons.ParseClientType(configs)
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return status.Error(codes.InvalidArgument, err.Error())
	}

	klog.V(5).Infof("mounting %q at %q", clientType, targetPath)
	switch clientType {
	case commons.IrodsFuseClientType:
		err = irods.Mount(irodsfsdClient, volID, configs, mountOptions, targetPath)
	case commons.WebdavClientType:
		err = webdav.Mount(irodsfsdClient, volID, configs, mountOptions, targetPath)
	case commons.NfsClientType:
		err = nfs.Mount(irodsfsdClient, volID, configs, mountOptions, targetPath)
	default:
		err = status.Errorf(codes.Internal, "unsupported client type %q", clientType)
	}
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return err
	}

	commons.IncreaseCounterForVolumeMount()
	commons.IncreaseCounterForActiveVolumeMount()
	return nil
}

// UnmountVolume records a daemon unmount request. The daemon owns actual
// cleanup, so this returns after the request has been accepted.
func UnmountVolume(irodsfsdClient *irodsfsd_client.MountServiceClient, volID string) error {
	if irodsfsdClient == nil {
		commons.IncreaseCounterForVolumeUnmountFailures()
		return status.Error(codes.FailedPrecondition, "irodsfsd client is not configured")
	}

	_, err := irodsfsdClient.Unmount(volID)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	if err != nil {
		commons.IncreaseCounterForVolumeUnmountFailures()
		code := status.Code(err)
		if code == codes.Unknown {
			code = codes.Internal
		}
		return status.Errorf(code, "request volume unmount %q: %v", volID, err)
	}

	commons.IncreaseCounterForVolumeUnmount()
	commons.DecreaseCounterForActiveVolumeMount()
	return nil
}
