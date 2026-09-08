package nfs

import (
	irodsfsd_client "github.com/cyverse/irodsfsd/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mount asks irodsfsd to mount an NFS export and waits until the mount is
// usable.
func Mount(irodsfsdClient *irodsfsd_client.MountServiceClient, volID string, configs map[string]string, mountOptions []string, targetPath string) error {
	if irodsfsdClient == nil {
		return status.Error(codes.FailedPrecondition, "irodsfsd client is not configured")
	}

	connectionInfo, err := GetConnectionInfo(configs)
	if err != nil {
		return err
	}

	mount, err := irodsfsdClient.MountWithID(volID, connectionInfo.MakeMountConfig(targetPath, mountOptions), true)
	if err != nil {
		return daemonStatusError("request NFS mount", volID, err)
	}
	if mount == nil {
		return status.Errorf(codes.Internal, "irodsfsd returned no mount for %q", volID)
	}
	return nil
}

// Unmount records an NFS unmount request with irodsfsd. Cleanup proceeds
// asynchronously after the daemon accepts this request.
func Unmount(irodsfsdClient *irodsfsd_client.MountServiceClient, volID string) error {
	if irodsfsdClient == nil {
		return status.Error(codes.FailedPrecondition, "irodsfsd client is not configured")
	}

	_, err := irodsfsdClient.Unmount(volID)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	if err != nil {
		return daemonStatusError("request NFS unmount", volID, err)
	}
	return nil
}

func daemonStatusError(operation string, volID string, err error) error {
	code := status.Code(err)
	if code == codes.Unknown {
		code = codes.Internal
	}
	return status.Errorf(code, "%s %q: %v", operation, volID, err)
}
