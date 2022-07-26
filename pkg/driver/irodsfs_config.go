package driver

import "time"

const (
	PortDefault                     int           = 1247
	ReadAheadMaxDefault             int           = 1024 * 64 // 64KB
	ConnectionMaxDefault            int           = 10
	OperationTimeoutDefault         time.Duration = 5 * time.Minute
	ConnectionLifespanDefault       time.Duration = 1 * time.Hour
	ConnectionIdleTimeoutDefault    time.Duration = 5 * time.Minute
	MetadataCacheTimeoutDefault     time.Duration = 5 * time.Minute
	MetadataCacheCleanupTimeDefault time.Duration = 5 * time.Minute
	BufferSizeMaxDefault            int64         = 1024 * 1024 * 64 // 64MB
)

// PathMapping ...
type IRODSFSPathMapping struct {
	IRODSPath           string `yaml:"irods_path" json:"irods_path"`
	MappingPath         string `yaml:"mapping_path" json:"mapping_path"`
	ResourceType        string `yaml:"resource_type" json:"resource_type"` // file or dir
	ReadOnly            bool   `yaml:"read_only" json:"read_only"`
	CreateDir           bool   `yaml:"create_dir" json:"create_dir"`
	IgnoreNotExistError bool   `yaml:"ignore_not_exist_error" json:"ignore_not_exist_error"`
}

type IRODSFSConfig struct {
	Host              string               `yaml:"host"`
	Port              int                  `yaml:"port"`
	ProxyUser         string               `yaml:"proxy_user"`
	ClientUser        string               `yaml:"client_user"`
	Zone              string               `yaml:"zone"`
	Password          string               `yaml:"password"`
	Resource          string               `yaml:"resource,omitempty"`
	PathMappings      []IRODSFSPathMapping `yaml:"path_mappings"`
	NoPermissionCheck bool                 `yaml:"no_permission_check"`
	UID               int                  `yaml:"uid"`
	GID               int                  `yaml:"gid"`
	SystemUser        string               `yaml:"system_user"`

	TempRootPath string `yaml:"temp_root_path,omitempty"`

	PoolEndpoint string `yaml:"pool_endpoint,omitempty"`

	ReadAheadMax     int           `yaml:"read_ahead_max"`
	OperationTimeout time.Duration `yaml:"operation_timeout"`

	ConnectionLifespan       time.Duration `yaml:"connection_lifespan"`
	ConnectionIdleTimeout    time.Duration `yaml:"connection_idle_timeout"`
	ConnectionMax            int           `yaml:"connection_max"`
	MetadataCacheTimeout     time.Duration `yaml:"metadata_cache_timeout"`
	MetadataCacheCleanupTime time.Duration `yaml:"metadata_cache_cleanup_time"`

	LogPath    string `yaml:"log_path,omitempty"`
	MonitorURL string `yaml:"monitor_url,omitempty"`

	Profile            bool `yaml:"profile,omitempty"`
	ProfileServicePort int  `yaml:"profile_service_port,omitempty"`

	AllowOther bool `yaml:"allow_other,omitempty"`
}

// NewDefaultIRODSFSConfig creates default IRODSFSConfig
func NewDefaultIRODSFSConfig() *IRODSFSConfig {
	return &IRODSFSConfig{
		Port:              PortDefault,
		PathMappings:      []IRODSFSPathMapping{},
		NoPermissionCheck: false,
		UID:               -1,
		GID:               -1,
		SystemUser:        "",

		TempRootPath: "",

		PoolEndpoint: "",

		ReadAheadMax:             ReadAheadMaxDefault,
		OperationTimeout:         OperationTimeoutDefault,
		ConnectionLifespan:       ConnectionLifespanDefault,
		ConnectionIdleTimeout:    ConnectionIdleTimeoutDefault,
		ConnectionMax:            ConnectionMaxDefault,
		MetadataCacheTimeout:     MetadataCacheTimeoutDefault,
		MetadataCacheCleanupTime: MetadataCacheCleanupTimeDefault,

		LogPath:            "",
		MonitorURL:         "",
		Profile:            false,
		ProfileServicePort: 0,
		AllowOther:         true,
	}
}
