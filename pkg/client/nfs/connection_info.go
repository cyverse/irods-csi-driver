package nfs

import (
	"path"
	"strconv"
	"strings"

	common "github.com/cyverse/irods-csi-driver/pkg/commons"
	irodsfsd_client "github.com/cyverse/irodsfsd/client"
	"github.com/cyverse/irodsfsd/service/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NFSConnectionInfo class
type NFSConnectionInfo struct {
	Hostname string
	Port     int
	Path     string
	ReadOnly bool
}

// MakeMountConfig converts the validated NFS settings into a daemon API
// configuration.
func (connInfo *NFSConnectionInfo) MakeMountConfig(targetPath string, mountOptions []string) *irodsfsd_client.MountConfig {
	return &irodsfsd_client.MountConfig{
		MountPath:    targetPath,
		ReadOnly:     connInfo.ReadOnly || hasReadOnlyOption(mountOptions),
		MountOptions: append([]string(nil), mountOptions...),
		ClientConfig: &api.MountConfig_Nfs{Nfs: &api.NFSConfig{
			Host: connInfo.Hostname,
			Port: int32(connInfo.Port),
			Path: connInfo.Path,
		}},
	}
}

func hasReadOnlyOption(mountOptions []string) bool {
	for _, option := range mountOptions {
		if strings.TrimSpace(option) == "ro" {
			return true
		}
	}
	return false
}

func getConnectionInfoFromMap(params map[string]string, connInfo *NFSConnectionInfo) error {
	for k, v := range params {
		switch common.NormalizeConfigKey(k) {
		case common.NormalizeConfigKey("host"), common.NormalizeConfigKey("hostname"):
			connInfo.Hostname = v
		case common.NormalizeConfigKey("port"):
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %v", k, err)
			}
			connInfo.Port = p
		case common.NormalizeConfigKey("path"):
			connInfo.Path = v
		case common.NormalizeConfigKey("read_only"):
			readOnly, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "argument %q must be a boolean: %v", k, err)
			}
			connInfo.ReadOnly = readOnly
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo returns NFSConnectionInfo value from param map
func GetConnectionInfo(configs map[string]string) (*NFSConnectionInfo, error) {
	connInfo := NFSConnectionInfo{}

	err := getConnectionInfoFromMap(configs, &connInfo)
	if err != nil {
		return nil, err
	}

	if len(connInfo.Hostname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host is empty")
	}

	if !path.IsAbs(connInfo.Path) {
		return nil, status.Errorf(codes.InvalidArgument, "argument path %q must be absolute", connInfo.Path)
	}

	if connInfo.Port == 0 {
		connInfo.Port = 2049
	}
	if connInfo.Port < 0 || connInfo.Port > 65535 {
		return nil, status.Errorf(codes.InvalidArgument, "argument port %d must be between 1 and 65535", connInfo.Port)
	}

	return &connInfo, nil
}
