package irods

import (
	"encoding/json"
	"path/filepath"
	"strconv"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	client_common "github.com/cyverse/irods-csi-driver/pkg/client/common"
	irodsfs_common_vpath "github.com/cyverse/irodsfs-common/vpath"
	irodsfs_commons "github.com/cyverse/irodsfs/commons"

	"github.com/cyverse/irods-csi-driver/pkg/common"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	irodsfsAnonymousUser       string = "anonymous"
	irodsfsDefaultMountTimeout int    = 60
)

// IRODSFSConnectionInfo class
type IRODSFSConnectionInfo struct {
	irodsfs_commons.Config

	MountTimeout    int
	OverlayFS       bool
	OverlayFSDriver OverlayFSDriverType
}

// NewIRODSFSConnectionInfo creates a new IRODSFSConnectionInfo with default
func NewIRODSFSConnectionInfo() IRODSFSConnectionInfo {
	connInfo := IRODSFSConnectionInfo{}
	connInfo.Config = *irodsfs_commons.NewDefaultConfig()

	connInfo.MountTimeout = irodsfsDefaultMountTimeout
	connInfo.OverlayFS = false
	connInfo.OverlayFSDriver = OverlayDriverType

	return connInfo
}

// SetAnonymousUser sets anonymous user
func (connInfo *IRODSFSConnectionInfo) SetAnonymousUser() {
	connInfo.Username = irodsfsAnonymousUser
}

// IsAnonymousUser checks if the user is anonymous
func (connInfo *IRODSFSConnectionInfo) IsAnonymousUser() bool {
	return connInfo.Username == irodsfsAnonymousUser
}

// IsValidClientUser checks if the client user is valid
func (connInfo *IRODSFSConnectionInfo) IsValidClientUser() bool {
	if len(connInfo.Username) > 0 && (connInfo.Username != irodsfsAnonymousUser) {
		// proxy is on
		return connInfo.ClientUsername == irodsfsAnonymousUser
	}
	return false
}

func getConnectionInfoFromMap(params map[string]string, connInfo *IRODSFSConnectionInfo) error {
	for k, v := range params {
		switch common.NormalizeConfigKey(k) {
		case common.NormalizeConfigKey("irods_authentication_scheme"), common.NormalizeConfigKey("authentication_scheme"), common.NormalizeConfigKey("auth_scheme"):
			connInfo.AuthenticationScheme = v
		case common.NormalizeConfigKey("irods_client_server_negotiation"), common.NormalizeConfigKey("client_server_negotiation"):
			connInfo.ClientServerNegotiation = v
		case common.NormalizeConfigKey("irods_client_server_policy"), common.NormalizeConfigKey("client_server_negotiation_policy"), common.NormalizeConfigKey("cs_negotiation_policy"):
			connInfo.ClientServerPolicy = v
		case common.NormalizeConfigKey("irods_host"), common.NormalizeConfigKey("hostname"), common.NormalizeConfigKey("host"):
			connInfo.Host = v
		case common.NormalizeConfigKey("irods_port"), common.NormalizeConfigKey("port"):
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %v", k, err)
			}
			connInfo.Port = p
		case common.NormalizeConfigKey("irods_zone_name"), common.NormalizeConfigKey("zone_name"), common.NormalizeConfigKey("zone"):
			connInfo.ZoneName = v
		case common.NormalizeConfigKey("irods_client_zone_name"), common.NormalizeConfigKey("client_zone_name"), common.NormalizeConfigKey("client_zone"):
			connInfo.ClientZoneName = v
		case common.NormalizeConfigKey("irods_user_name"), common.NormalizeConfigKey("user"), common.NormalizeConfigKey("user_name"):
			connInfo.Username = v
		case common.NormalizeConfigKey("irods_client_user_name"), common.NormalizeConfigKey("client_user"), common.NormalizeConfigKey("client_user_name"):
			connInfo.ClientUsername = v
		case common.NormalizeConfigKey("irods_default_resource"), common.NormalizeConfigKey("default_resource"), common.NormalizeConfigKey("resource"):
			connInfo.DefaultResource = v
		case common.NormalizeConfigKey("irods_encryption_algorithm"), common.NormalizeConfigKey("encryption_algorithm"):
			connInfo.EncryptionAlgorithm = v
		case common.NormalizeConfigKey("irods_encryption_key_size"), common.NormalizeConfigKey("encryption_key_size"):
			s, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.EncryptionKeySize = s
		case common.NormalizeConfigKey("irods_encryption_salt_size"), common.NormalizeConfigKey("encryption_salt_size"):
			s, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.EncryptionSaltSize = s
		case common.NormalizeConfigKey("irods_encryption_num_hash_rounds"), common.NormalizeConfigKey("encryption_num_hash_rounds"), common.NormalizeConfigKey("hash_rounds"):
			s, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.EncryptionNumHashRounds = s
		case common.NormalizeConfigKey("irods_ssl_ca_certificate_file"), common.NormalizeConfigKey("ca_certificate_file"):
			connInfo.SSLCACertificateFile = v
		case common.NormalizeConfigKey("irods_ssl_ca_certificate_path"), common.NormalizeConfigKey("ca_certificate_path"):
			connInfo.SSLCACertificatePath = v
		case common.NormalizeConfigKey("irods_ssl_verify_server"), common.NormalizeConfigKey("verify_server"):
			connInfo.SSLVerifyServer = v
		case common.NormalizeConfigKey("irods_user_password"), common.NormalizeConfigKey("user_password"), common.NormalizeConfigKey("password"):
			connInfo.Password = v
		case common.NormalizeConfigKey("irods_ssl_server_name"), common.NormalizeConfigKey("ssl_server_name"):
			connInfo.SSLServerName = v
		case common.NormalizeConfigKey("path_mappings"), common.NormalizeConfigKey("path_mapping_json"):
			connInfo.PathMappings = []irodsfs_common_vpath.VPathMapping{}
			err := json.Unmarshal([]byte(v), &connInfo.PathMappings)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case common.NormalizeConfigKey("path"):
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
		case common.NormalizeConfigKey("read_ahead_max"):
			ram, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.ReadAheadMax = ram
		case common.NormalizeConfigKey("no_permission_check"):
			npc, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.NoPermissionCheck = npc
		case common.NormalizeConfigKey("no_set_xattr"):
			nsx, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.NoSetXattr = nsx
		case common.NormalizeConfigKey("uid"), common.NormalizeConfigKey("user_id"):
			u, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid uid number - %v", k, err)
			}
			connInfo.UID = u
		case common.NormalizeConfigKey("gid"), common.NormalizeConfigKey("group_id"):
			g, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid gid number - %v", k, err)
			}
			connInfo.GID = g
		case common.NormalizeConfigKey("system_user"):
			connInfo.SystemUser = v
		case common.NormalizeConfigKey("metadata_connection"), common.NormalizeConfigKey("metadata_connection_json"):
			connInfo.MetadataConnection = irodsclient_fs.NewDefaultMetadataConnectionConfig()
			err := json.Unmarshal([]byte(v), &connInfo.MetadataConnection)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case common.NormalizeConfigKey("io_connection"), common.NormalizeConfigKey("io_connection_json"):
			connInfo.IOConnection = irodsclient_fs.NewDefaultIOConnectionConfig()
			err := json.Unmarshal([]byte(v), &connInfo.IOConnection)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case common.NormalizeConfigKey("cache"), common.NormalizeConfigKey("cache_json"):
			connInfo.Cache = irodsclient_fs.NewDefaultCacheConfig()
			err := json.Unmarshal([]byte(v), &connInfo.Cache)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case common.NormalizeConfigKey("pool_endpoint"):
			connInfo.PoolEndpoint = v
		case common.NormalizeConfigKey("enable_profile"), common.NormalizeConfigKey("profile"):
			pb, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.Profile = pb
		case common.NormalizeConfigKey("profile_service_port"), common.NormalizeConfigKey("profile_port"):
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %v", k, err)
			}
			connInfo.ProfileServicePort = p
		case common.NormalizeConfigKey("enable_overlayfs"), common.NormalizeConfigKey("overlayfs"):
			ob, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.OverlayFS = ob
		case common.NormalizeConfigKey("overlayfs_driver"):
			connInfo.OverlayFSDriver = GetOverlayFSDriverType(v)
		case common.NormalizeConfigKey("mount_timeout"):
			t, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.MountTimeout = t
		case common.NormalizeConfigKey("debug"):
			debug, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.Debug = debug
		case common.NormalizeConfigKey("read_only"):
			ro, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.Readonly = ro
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo extracts IRODSFSConnectionInfo value from param map
func GetConnectionInfo(configs map[string]string) (*IRODSFSConnectionInfo, error) {
	connInfo := NewIRODSFSConnectionInfo()

	err := getConnectionInfoFromMap(configs, &connInfo)
	if err != nil {
		return nil, err
	}

	// correct
	connInfo.FixAuthConfiguration()
	err = connInfo.FixSystemUserConfiguration()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Argument systemuser/uid/gid are not valid - %q", err.Error())
	}

	// validate
	if len(connInfo.OverlayFSDriver) == 0 {
		connInfo.OverlayFSDriver = OverlayDriverType
	}

	if len(connInfo.Username) == 0 && len(connInfo.ClientUsername) == 0 {
		connInfo.SetAnonymousUser()
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && !connInfo.IsAnonymousUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument password must be given")
	}

	if connInfo.IsValidClientUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument client username must be a non-anonymous user")
	}

	if client_common.GetConfigEnforceProxyAccess(configs) {
		// we don't allow anonymous user
		if connInfo.IsAnonymousUser() {
			return nil, status.Error(codes.InvalidArgument, "Argument user must be a non-anonymous user")
		}

		if len(connInfo.ClientUsername) == 0 {
			return nil, status.Error(codes.InvalidArgument, "Argument client username must be given")
		}

		if connInfo.Username == connInfo.ClientUsername {
			return nil, status.Errorf(codes.InvalidArgument, "Argument client username cannot be the same as user - user %q, client user %q", connInfo.Username, connInfo.ClientUsername)
		}
	}

	if len(connInfo.Host) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host must be given")
	}

	if connInfo.Port <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument port must be given")
	}

	if len(connInfo.Username) == 0 && len(connInfo.ClientUsername) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument username or client username must be given")
	}

	if len(connInfo.ZoneName) == 0 && len(connInfo.ClientZoneName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument zone name or client zone name must be given")
	}

	if connInfo.Profile && connInfo.ProfileServicePort <= 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument profile service port must be given")
	}

	if len(connInfo.PathMappings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument path or path mappings must be given")
	}

	err = irodsfs_common_vpath.ValidateVPathMappings(connInfo.PathMappings)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Path mapping is invalid - %q", err.Error())
	}

	if connInfo.UID < 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument uid must not be a negative value")
	}

	if connInfo.GID < 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument gid must not be a negative value")
	}

	if connInfo.ReadAheadMax < 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument read_ahead must not be a negative value")
	}

	if len(connInfo.PoolEndpoint) > 0 {
		_, err := common.ParsePoolServiceEndpoint(connInfo.PoolEndpoint)
		if err != nil {
			return nil, err
		}
	}

	whitelist := client_common.GetConfigMountPathWhitelist(configs)
	for _, mapping := range connInfo.PathMappings {
		if !mounter.IsMountPathAllowed(whitelist, mapping.IRODSPath) {
			return nil, status.Errorf(codes.InvalidArgument, "Argument path %q is not allowed to mount by the whitelist", mapping.IRODSPath)
		}
	}

	if connInfo.MountTimeout <= 0 {
		connInfo.MountTimeout = 60
	}

	return &connInfo, nil
}
