package irods

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cyverse/irods-csi-driver/pkg/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IRODSFSConnectionInfo class
type IRODSFSConnectionInfo struct {
	Hostname          string
	Port              int
	Zone              string
	User              string
	Password          string
	ClientUser        string // if this field has a value, user and password fields have proxy user info
	Resource          string
	PoolEndpoint      string
	MonitorURL        string
	PathMappings      []IRODSFSPathMapping
	NoPermissionCheck bool
	UID               int
	GID               int
	SystemUser        string
	MountTimeout      int
	Profile           bool
	ProfilePort       int
}

func getConnectionInfoFromMap(params map[string]string, connInfo *IRODSFSConnectionInfo) error {
	for k, v := range params {
		switch strings.ToLower(k) {
		case "user":
			connInfo.User = v
		case "password":
			connInfo.Password = v
		case "client_user", "clientuser":
			// for proxy
			connInfo.ClientUser = v
		case "host":
			connInfo.Hostname = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			connInfo.Port = p
		case "zone":
			connInfo.Zone = v
		case "resource":
			connInfo.Resource = v
		case "path":
			if !filepath.IsAbs(v) {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}

			// mount a collection
			connInfo.PathMappings = []IRODSFSPathMapping{
				{
					IRODSPath:    v,
					MappingPath:  "/",
					ResourceType: "dir",
				},
			}
		case "pool_endpoint", "poolendpoint":
			pe, err := common.ParsePoolServiceEndpoint(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid pool endpoint - %s", k, err)
			}

			connInfo.PoolEndpoint = pe
		case "profile":
			pb, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %s", k, err)
			}
			connInfo.Profile = pb
		case "profile_port", "profileport":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			connInfo.ProfilePort = p
		case "monitorurl", "monitor_url":
			connInfo.MonitorURL = v
		case "path_mapping_json", "pathmappingjson":
			connInfo.PathMappings = []IRODSFSPathMapping{}
			err := json.Unmarshal([]byte(v), &connInfo.PathMappings)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %s", k, err)
			}
		case "no_permission_check", "nopermissioncheck":
			npc, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %s", k, err)
			}
			connInfo.NoPermissionCheck = npc
		case "uid":
			u, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid uid number - %s", k, err)
			}
			connInfo.UID = u
		case "gid":
			g, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid gid number - %s", k, err)
			}
			connInfo.GID = g
		case "system_user", "systemuser", "sys_user", "sysuser":
			connInfo.SystemUser = v
		case "mount_timeout", "mounttimeout":
			t, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %s", k, err)
			}
			connInfo.MountTimeout = t
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo extracts IRODSFSConnectionInfo value from param map
func GetConnectionInfo(poolEndpoint string, params map[string]string, secrets map[string]string) (*IRODSFSConnectionInfo, error) {
	connInfo := IRODSFSConnectionInfo{}
	connInfo.UID = -1
	connInfo.GID = -1

	err := getConnectionInfoFromMap(secrets, &connInfo)
	if err != nil {
		return nil, err
	}

	err = getConnectionInfoFromMap(params, &connInfo)
	if err != nil {
		return nil, err
	}

	if len(connInfo.User) == 0 {
		connInfo.User = "anonymous"
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && connInfo.User != "anonymous" {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(connInfo.ClientUser) == 0 {
		connInfo.ClientUser = connInfo.User
	}

	if len(connInfo.Hostname) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host is empty")
	}

	if len(connInfo.Zone) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument zone is empty")
	}

	// port is optional
	if connInfo.Port <= 0 {
		// default
		connInfo.Port = 1247
	}

	// profile port is optional
	if connInfo.ProfilePort <= 0 {
		// default
		connInfo.ProfilePort = 11021
	}

	if len(connInfo.MonitorURL) > 0 {
		// check
		_, err := url.ParseRequestURI(connInfo.MonitorURL)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid monitor URL - %s", connInfo.MonitorURL)
		}
	}

	if connInfo.MountTimeout <= 0 {
		connInfo.MountTimeout = 300
	}

	return &connInfo, nil
}
