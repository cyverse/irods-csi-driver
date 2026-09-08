package irods

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsfs_common_vpath "github.com/cyverse/irodsfs-common/irods/vpath"
	irodsfs_commons "github.com/cyverse/irodsfs/commons"

	commons "github.com/cyverse/irods-csi-driver/pkg/commons"
	irodsfsd_client "github.com/cyverse/irodsfsd/client"
	"github.com/cyverse/irodsfsd/service/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	irodsfsAnonymousUser string = "anonymous"
)

// IRODSFSConnectionInfo class
type IRODSFSConnectionInfo struct {
	irodsfs_commons.Config
}

// NewIRODSFSConnectionInfo creates a new IRODSFSConnectionInfo with default
func NewIRODSFSConnectionInfo() IRODSFSConnectionInfo {
	connInfo := IRODSFSConnectionInfo{
		Config: *irodsfs_commons.NewDefaultConfig(),
	}

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
		return connInfo.ClientUsername != irodsfsAnonymousUser
	}
	return true
}

// MakeMountConfig converts validated iRODS settings into the irodsfsd API
// configuration. It must never be logged because it can contain credentials.
func (connInfo *IRODSFSConnectionInfo) MakeMountConfig(targetPath string, mountOptions []string) *irodsfsd_client.MountConfig {
	account := &api.Account{
		IrodsHost:     connInfo.Host,
		IrodsPort:     int32(connInfo.Port),
		IrodsZoneName: connInfo.ZoneName,
		IrodsUserName: connInfo.Username,
	}
	setAccountOptions(account, connInfo)
	mappings := make([]*api.PathMapping, 0, len(connInfo.PathMappings))
	for _, mapping := range connInfo.PathMappings {
		mappings = append(mappings, &api.PathMapping{
			IrodsPath:           mapping.IRODSPath,
			MappingPath:         mapping.MappingPath,
			ResourceType:        string(mapping.ResourceType),
			ReadOnly:            mapping.ReadOnly,
			CreateDir:           mapping.CreateDir,
			IgnoreNotExistError: mapping.IgnoreNotExistError,
		})
	}
	irodsfsConfig := &api.IRODSFSConfig{
		Account:            account,
		PathMappings:       mappings,
		FuseOptions:        append([]string(nil), connInfo.FuseOptions...),
		MetadataConnection: makeConnectionConfig(connInfo.MetadataConnection),
		IoConnection:       makeConnectionConfig(connInfo.IOConnection),
		Cache:              makeCacheConfig(connInfo.Cache),
		Debug:              connInfo.Debug,
	}
	if connInfo.ReadAheadMax != 0 {
		irodsfsConfig.ReadAheadMax = proto.Int32(int32(connInfo.ReadAheadMax))
	}
	if connInfo.ReadWriteMax != 0 {
		irodsfsConfig.ReadWriteMax = proto.Int32(int32(connInfo.ReadWriteMax))
	}
	if connInfo.UID >= 0 {
		irodsfsConfig.Uid = proto.Int32(int32(connInfo.UID))
	}
	if connInfo.GID >= 0 {
		irodsfsConfig.Gid = proto.Int32(int32(connInfo.GID))
	}
	if connInfo.SystemUser != "" {
		irodsfsConfig.SystemUser = proto.String(connInfo.SystemUser)
	}
	if connInfo.PoolEndpoint != "" {
		irodsfsConfig.PoolEndpoint = proto.String(connInfo.PoolEndpoint)
	}
	return &irodsfsd_client.MountConfig{
		MountPath:    targetPath,
		ReadOnly:     connInfo.Readonly || hasReadOnlyOption(mountOptions),
		MountOptions: daemonMountOptions(mountOptions),
		ClientConfig: &api.MountConfig_Irodsfs{
			Irodsfs: irodsfsConfig,
		},
	}
}

func setAccountOptions(account *api.Account, connInfo *IRODSFSConnectionInfo) {
	if connInfo.AuthenticationScheme != "" {
		account.IrodsAuthenticationScheme = proto.String(connInfo.AuthenticationScheme)
	}
	if connInfo.ClientServerNegotiation != "" {
		account.IrodsClientServerNegotiation = proto.Bool(connInfo.ClientServerNegotiation != "off")
	}
	if connInfo.ClientServerPolicy != "" {
		account.IrodsClientServerPolicy = proto.String(connInfo.ClientServerPolicy)
	}
	if connInfo.ClientZoneName != "" {
		account.IrodsClientZoneName = proto.String(connInfo.ClientZoneName)
	}
	if connInfo.ClientUsername != "" {
		account.IrodsClientUserName = proto.String(connInfo.ClientUsername)
	}
	if connInfo.DefaultResource != "" {
		account.IrodsDefaultResource = proto.String(connInfo.DefaultResource)
	}
	if connInfo.CurrentWorkingDir != "" {
		account.IrodsCwd = proto.String(connInfo.CurrentWorkingDir)
	}
	if connInfo.Home != "" {
		account.IrodsHome = proto.String(connInfo.Home)
	}
	if connInfo.DefaultHashScheme != "" {
		account.IrodsDefaultHashScheme = proto.String(connInfo.DefaultHashScheme)
	}
	if connInfo.MatchHashPolicy != "" {
		account.IrodsMatchHashPolicy = proto.String(connInfo.MatchHashPolicy)
	}
	if connInfo.EncryptionAlgorithm != "" {
		account.IrodsEncryptionAlgorithm = proto.String(connInfo.EncryptionAlgorithm)
	}
	if connInfo.EncryptionKeySize != 0 {
		account.IrodsEncryptionKeySize = proto.Int32(int32(connInfo.EncryptionKeySize))
	}
	if connInfo.EncryptionSaltSize != 0 {
		account.IrodsEncryptionSaltSize = proto.Int32(int32(connInfo.EncryptionSaltSize))
	}
	if connInfo.EncryptionNumHashRounds != 0 {
		account.IrodsEncryptionNumHashRounds = proto.Int32(int32(connInfo.EncryptionNumHashRounds))
	}
	if connInfo.SSLCACertificateFile != "" {
		account.IrodsSslCaCertificateFile = proto.String(connInfo.SSLCACertificateFile)
	}
	if connInfo.SSLCACertificatePath != "" {
		account.IrodsSslCaCertificatePath = proto.String(connInfo.SSLCACertificatePath)
	}
	if connInfo.SSLVerifyServer != "" {
		account.IrodsSslVerifyServer = proto.String(connInfo.SSLVerifyServer)
	}
	if connInfo.SSLCertificateChainFile != "" {
		account.IrodsSslCertificateChainFile = proto.String(connInfo.SSLCertificateChainFile)
	}
	if connInfo.SSLCertificateKeyFile != "" {
		account.IrodsSslCertificateKeyFile = proto.String(connInfo.SSLCertificateKeyFile)
	}
	if connInfo.SSLDHParamsFile != "" {
		account.IrodsSslDhParamsFile = proto.String(connInfo.SSLDHParamsFile)
	}
	if connInfo.Password != "" {
		account.IrodsUserPassword = proto.String(connInfo.Password)
	}
	if connInfo.Ticket != "" {
		account.IrodsTicket = proto.String(connInfo.Ticket)
	}
	if connInfo.PAMToken != "" {
		account.IrodsPamToken = proto.String(connInfo.PAMToken)
	}
	if connInfo.PAMTTL != 0 {
		account.IrodsPamTtl = proto.Int32(int32(connInfo.PAMTTL))
	}
	if connInfo.SSLServerName != "" {
		account.IrodsSslServerName = proto.String(connInfo.SSLServerName)
	}
}

func hasReadOnlyOption(options []string) bool {
	for _, option := range options {
		if strings.TrimSpace(option) == "ro" {
			return true
		}
	}
	return false
}

func daemonMountOptions(options []string) []string {
	filtered := make([]string, 0, len(options))
	for _, option := range options {
		if option != "allow_other" && option != "config=-" && !strings.HasPrefix(option, "mounttimeout=") {
			filtered = append(filtered, option)
		}
	}
	return filtered
}

func makeConnectionConfig(config irodsclient_fs.ConnectionConfig) *api.ConnectionConfig {
	result := &api.ConnectionConfig{}
	if config.CreationTimeout != 0 {
		result.CreationTimeout = durationpb.New(time.Duration(config.CreationTimeout))
	}
	if config.InitNumber != 0 {
		result.InitNumber = proto.Int32(int32(config.InitNumber))
	}
	if config.MaxNumber != 0 {
		result.MaxNumber = proto.Int32(int32(config.MaxNumber))
	}
	if config.MaxIdleNumber != 0 {
		result.MaxIdleNumber = proto.Int32(int32(config.MaxIdleNumber))
	}
	if config.Lifespan != 0 {
		result.Lifespan = durationpb.New(time.Duration(config.Lifespan))
	}
	if config.IdleTimeout != 0 {
		result.IdleTimeout = durationpb.New(time.Duration(config.IdleTimeout))
	}
	if config.OperationTimeout != 0 {
		result.OperationTimeout = durationpb.New(time.Duration(config.OperationTimeout))
	}
	if config.LongOperationTimeout != 0 {
		result.LongOperationTimeout = durationpb.New(time.Duration(config.LongOperationTimeout))
	}
	if config.TcpBufferSize != 0 {
		result.TcpBufferSize = proto.Int32(int32(config.TcpBufferSize))
	}
	if config.WaitConnection {
		result.WaitConnection = proto.Bool(true)
	}
	return result
}

func makeCacheConfig(config irodsclient_fs.CacheConfig) *api.CacheConfig {
	settings := make([]*api.MetadataCacheTimeoutSetting, 0, len(config.MetadataTimeoutSettings))
	for _, setting := range config.MetadataTimeoutSettings {
		settings = append(settings, &api.MetadataCacheTimeoutSetting{Path: setting.Path, Timeout: durationpb.New(time.Duration(setting.Timeout)), Inherit: proto.Bool(setting.Inherit)})
	}
	result := &api.CacheConfig{MetadataTimeoutSettings: settings}
	if config.StartNewTransaction {
		result.StartNewTransaction = proto.Bool(true)
	}
	if config.Backend != nil {
		backend := &api.CacheBackendConfig{Type: string(config.Backend.Type)}
		if config.Backend.Memory != nil {
			backend.Memory = &api.MemoryBackendConfig{CleanupInterval: durationpb.New(config.Backend.Memory.CleanupInterval), DefaultTtl: durationpb.New(config.Backend.Memory.DefaultTTL)}
		}
		if config.Backend.Ristretto != nil {
			backend.Ristretto = &api.RistrettoBackendConfig{MaxEntries: proto.Int64(config.Backend.Ristretto.MaxEntries), MaxCost: proto.Int64(config.Backend.Ristretto.MaxCost), BufferItems: proto.Int64(config.Backend.Ristretto.BufferItems), DefaultTtl: durationpb.New(config.Backend.Ristretto.DefaultTTL)}
		}
		if config.Backend.Redis != nil {
			backend.Redis = &api.RedisBackendConfig{Address: proto.String(config.Backend.Redis.Address), Db: proto.Int32(int32(config.Backend.Redis.DB)), Password: proto.String(config.Backend.Redis.Password), PoolSize: proto.Int32(int32(config.Backend.Redis.PoolSize)), KeyPrefix: proto.String(config.Backend.Redis.KeyPrefix), ConnectTimeout: durationpb.New(config.Backend.Redis.ConnectTimeout), CommandTimeout: durationpb.New(config.Backend.Redis.CommandTimeout), DefaultTtl: durationpb.New(config.Backend.Redis.DefaultTTL), EnableAccountIsolation: proto.Bool(config.Backend.Redis.EnableAccountIsolation)}
		}
		result.Backend = backend
	}
	return result
}

func getConnectionInfoFromMap(params map[string]string, connInfo *IRODSFSConnectionInfo) error {
	for k, v := range params {
		switch commons.NormalizeConfigKey(k) {
		case commons.NormalizeConfigKey("irods_authentication_scheme"), commons.NormalizeConfigKey("authentication_scheme"), commons.NormalizeConfigKey("auth_scheme"):
			connInfo.AuthenticationScheme = v
		case commons.NormalizeConfigKey("irods_client_server_negotiation"), commons.NormalizeConfigKey("client_server_negotiation"):
			connInfo.ClientServerNegotiation = v
		case commons.NormalizeConfigKey("irods_client_server_policy"), commons.NormalizeConfigKey("client_server_negotiation_policy"), commons.NormalizeConfigKey("cs_negotiation_policy"):
			connInfo.ClientServerPolicy = v
		case commons.NormalizeConfigKey("irods_host"), commons.NormalizeConfigKey("hostname"), commons.NormalizeConfigKey("host"):
			connInfo.Host = v
		case commons.NormalizeConfigKey("irods_port"), commons.NormalizeConfigKey("port"):
			p, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %v", k, err)
			}
			connInfo.Port = p
		case commons.NormalizeConfigKey("irods_zone_name"), commons.NormalizeConfigKey("zone_name"), commons.NormalizeConfigKey("zone"):
			connInfo.ZoneName = v
		case commons.NormalizeConfigKey("irods_client_zone_name"), commons.NormalizeConfigKey("client_zone_name"), commons.NormalizeConfigKey("client_zone"):
			connInfo.ClientZoneName = v
		case commons.NormalizeConfigKey("irods_user_name"), commons.NormalizeConfigKey("user"), commons.NormalizeConfigKey("user_name"):
			connInfo.Username = v
		case commons.NormalizeConfigKey("irods_client_user_name"), commons.NormalizeConfigKey("client_user"), commons.NormalizeConfigKey("client_user_name"):
			connInfo.ClientUsername = v
		case commons.NormalizeConfigKey("irods_default_resource"), commons.NormalizeConfigKey("default_resource"), commons.NormalizeConfigKey("resource"):
			connInfo.DefaultResource = v
		case commons.NormalizeConfigKey("irods_encryption_algorithm"), commons.NormalizeConfigKey("encryption_algorithm"):
			connInfo.EncryptionAlgorithm = v
		case commons.NormalizeConfigKey("irods_encryption_key_size"), commons.NormalizeConfigKey("encryption_key_size"):
			s, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.EncryptionKeySize = s
		case commons.NormalizeConfigKey("irods_encryption_salt_size"), commons.NormalizeConfigKey("encryption_salt_size"):
			s, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.EncryptionSaltSize = s
		case commons.NormalizeConfigKey("irods_encryption_num_hash_rounds"), commons.NormalizeConfigKey("encryption_num_hash_rounds"), commons.NormalizeConfigKey("hash_rounds"):
			s, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.EncryptionNumHashRounds = s
		case commons.NormalizeConfigKey("irods_ssl_ca_certificate_file"), commons.NormalizeConfigKey("ca_certificate_file"):
			connInfo.SSLCACertificateFile = v
		case commons.NormalizeConfigKey("irods_ssl_ca_certificate_path"), commons.NormalizeConfigKey("ca_certificate_path"):
			connInfo.SSLCACertificatePath = v
		case commons.NormalizeConfigKey("irods_ssl_verify_server"), commons.NormalizeConfigKey("verify_server"):
			connInfo.SSLVerifyServer = v
		case commons.NormalizeConfigKey("irods_user_password"), commons.NormalizeConfigKey("user_password"), commons.NormalizeConfigKey("password"):
			connInfo.Password = v
		case commons.NormalizeConfigKey("irods_ssl_server_name"), commons.NormalizeConfigKey("ssl_server_name"):
			connInfo.SSLServerName = v
		case commons.NormalizeConfigKey("path_mappings"), commons.NormalizeConfigKey("path_mapping_json"):
			connInfo.PathMappings = []irodsfs_common_vpath.VPathMapping{}
			err := json.Unmarshal([]byte(v), &connInfo.PathMappings)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case commons.NormalizeConfigKey("path"):
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
		case commons.NormalizeConfigKey("read_ahead_max"):
			ram, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid number - %v", k, err)
			}
			connInfo.ReadAheadMax = ram
		case commons.NormalizeConfigKey("uid"), commons.NormalizeConfigKey("user_id"):
			u, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid uid number - %v", k, err)
			}
			connInfo.UID = u
		case commons.NormalizeConfigKey("gid"), commons.NormalizeConfigKey("group_id"):
			g, err := strconv.Atoi(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid gid number - %v", k, err)
			}
			connInfo.GID = g
		case commons.NormalizeConfigKey("system_user"):
			connInfo.SystemUser = v
		case commons.NormalizeConfigKey("metadata_connection"), commons.NormalizeConfigKey("metadata_connection_json"):
			connInfo.MetadataConnection = irodsclient_fs.NewDefaultMetadataConnectionConfig()
			err := json.Unmarshal([]byte(v), &connInfo.MetadataConnection)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case commons.NormalizeConfigKey("io_connection"), commons.NormalizeConfigKey("io_connection_json"):
			connInfo.IOConnection = irodsclient_fs.NewDefaultIOConnectionConfig()
			err := json.Unmarshal([]byte(v), &connInfo.IOConnection)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case commons.NormalizeConfigKey("cache"), commons.NormalizeConfigKey("cache_json"):
			connInfo.Cache = irodsclient_fs.NewDefaultCacheConfig()
			err := json.Unmarshal([]byte(v), &connInfo.Cache)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid json string - %v", k, err)
			}
		case commons.NormalizeConfigKey("pool_endpoint"):
			connInfo.PoolEndpoint = v
		case commons.NormalizeConfigKey("debug"):
			debug, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a valid boolean string - %v", k, err)
			}
			connInfo.Debug = debug
		case commons.NormalizeConfigKey("read_only"):
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
	if len(connInfo.Username) == 0 && len(connInfo.ClientUsername) == 0 {
		connInfo.SetAnonymousUser()
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && !connInfo.IsAnonymousUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument password must be given")
	}

	if !connInfo.IsValidClientUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument client username must be a non-anonymous user")
	}

	if getConfigEnforceProxyAccess(configs) {
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
		_, _, err := commons.ParseServiceEndpoint(connInfo.PoolEndpoint)
		if err != nil {
			return nil, err
		}
	}

	whitelist := getConfigMountPathWhitelist(configs)
	for _, mapping := range connInfo.PathMappings {
		if !isMountPathAllowed(whitelist, mapping.IRODSPath) {
			return nil, status.Errorf(codes.InvalidArgument, "Argument path %q is not allowed to mount by the whitelist", mapping.IRODSPath)
		}
	}

	return &connInfo, nil
}

func isMountPathAllowed(whitelist []string, path string) bool {
	for _, allowedPath := range whitelist {
		relativePath, err := filepath.Rel(allowedPath, path)
		if err == nil && relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func getConfigEnforceProxyAccess(configs map[string]string) bool {
	enforce, err := strconv.ParseBool(configs[commons.NormalizeConfigKey("enforce_proxy_access")])
	return err == nil && enforce
}

func getConfigMountPathWhitelist(configs map[string]string) []string {
	whitelist := configs[commons.NormalizeConfigKey("mount_path_whitelist")]
	if whitelist == "" {
		return []string{"/"}
	}

	items := strings.Split(whitelist, ",")
	for index, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			item = "/"
		}
		items[index] = item
	}
	return items
}
