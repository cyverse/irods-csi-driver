package irods

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strconv"

	client_common "github.com/cyverse/irods-csi-driver/pkg/client/common"
	irodsfs_common_vpath "github.com/cyverse/irodsfs-common/vpath"

	"github.com/cyverse/irods-csi-driver/pkg/common"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	irodsfsAnonymousUser string = "anonymous"
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
	PathMappings      []irodsfs_common_vpath.VPathMapping
	NoPermissionCheck bool
	UID               int
	GID               int
	SystemUser        string
	MountTimeout      int
	Profile           bool
	ProfilePort       int
	OverlayFS         bool
	OverlayFSDriver   OverlayFSDriverType
}

// SetAnonymousUser sets anonymous user
func (connInfo *IRODSFSConnectionInfo) SetAnonymousUser() {
	connInfo.User = irodsfsAnonymousUser
}

// IsAnonymousUser checks if the user is anonymous
func (connInfo *IRODSFSConnectionInfo) IsAnonymousUser() bool {
	return connInfo.User == irodsfsAnonymousUser
}

// IsAnonymousClientUser checks if the client user is anonymous
func (connInfo *IRODSFSConnectionInfo) IsAnonymousClientUser() bool {
	return connInfo.ClientUser == irodsfsAnonymousUser
}

func getConnectionInfoFromMap(params map[string]string, connInfo *IRODSFSConnectionInfo) error {
	for k, v := range params {
		switch k {
		case "user":
			connInfo.User = v
		case "password":
			connInfo.Password = v
		case "clientuser":
			// for proxy
			connInfo.ClientUser = v
		case "host":
			connInfo.Hostname = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %v", k, err)
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
			connInfo.PathMappings = []irodsfs_common_vpath.VPathMapping{
				{
					IRODSPath:    v,
					MappingPath:  "/",
					ResourceType: "dir",
				},
			}
		case "poolendpoint":
			connInfo.PoolEndpoint = v
		case "profile":
			pb, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.Profile = pb
		case "profileport":
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %v", k, err)
			}
			connInfo.ProfilePort = p
		case "overlayfs":
			ob, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.OverlayFS = ob
		case "overlayfsdriver":
			connInfo.OverlayFSDriver = GetOverlayFSDriverType(v)
		case "monitorurl":
			connInfo.MonitorURL = v
		case "pathmappingjson":
			connInfo.PathMappings = []irodsfs_common_vpath.VPathMapping{}
			err := json.Unmarshal([]byte(v), &connInfo.PathMappings)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case "nopermissioncheck":
			npc, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.NoPermissionCheck = npc
		case "uid":
			u, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid uid number - %v", k, err)
			}
			connInfo.UID = u
		case "gid":
			g, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid gid number - %v", k, err)
			}
			connInfo.GID = g
		case "systemuser":
			connInfo.SystemUser = v
		case "mounttimeout":
			t, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.MountTimeout = t
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo extracts IRODSFSConnectionInfo value from param map
func GetConnectionInfo(configs map[string]string) (*IRODSFSConnectionInfo, error) {
	connInfo := IRODSFSConnectionInfo{}
	connInfo.UID = -1
	connInfo.GID = -1

	err := getConnectionInfoFromMap(configs, &connInfo)
	if err != nil {
		return nil, err
	}

	if len(connInfo.OverlayFSDriver) == 0 {
		connInfo.OverlayFSDriver = OverlayDriverType
	}

	if len(connInfo.User) == 0 {
		connInfo.SetAnonymousUser()
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && !connInfo.IsAnonymousUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if connInfo.IsAnonymousClientUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument clientUser must be a non-anonymous user")
	}

	if client_common.GetConfigEnforceProxyAccess(configs) {
		// we don't allow anonymous user
		if connInfo.IsAnonymousUser() {
			return nil, status.Error(codes.InvalidArgument, "Argument user must be a non-anonymous user")
		}

		if len(connInfo.ClientUser) == 0 {
			return nil, status.Error(codes.InvalidArgument, "Argument clientUser must be given")
		}

		if connInfo.User == connInfo.ClientUser {
			return nil, status.Errorf(codes.InvalidArgument, "Argument clientUser cannot be the same as user - user %q, clientUser %q", connInfo.User, connInfo.ClientUser)
		}
	} else {
		if len(connInfo.ClientUser) == 0 {
			connInfo.ClientUser = connInfo.User
		}
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

	if len(connInfo.PoolEndpoint) > 0 {
		_, err := common.ParsePoolServiceEndpoint(connInfo.PoolEndpoint)
		if err != nil {
			return nil, err
		}
	}

	if len(connInfo.MonitorURL) > 0 {
		// check
		_, err := url.ParseRequestURI(connInfo.MonitorURL)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid monitor URL %q", connInfo.MonitorURL)
		}
	}

	if len(connInfo.PathMappings) < 1 {
		return nil, status.Error(codes.InvalidArgument, "Argument path and path_mappings are empty, one must be given")
	}

	whitelist := client_common.GetConfigMountPathWhitelist(configs)
	for _, mapping := range connInfo.PathMappings {
		if !mounter.IsMountPathAllowed(whitelist, mapping.IRODSPath) {
			return nil, status.Errorf(codes.InvalidArgument, "Argument path %q is not allowed to mount", mapping.IRODSPath)
		}
	}

	if connInfo.MountTimeout <= 0 {
		connInfo.MountTimeout = 300
	}

	return &connInfo, nil
}
