package driver

import "time"

const (
	PortDefault                     int           = 1247
	ReadAheadMaxDefault             int           = 1024 * 64 // 64KB
	ConnectionMaxDefault            int           = 10
	OperationTimeoutDefault         time.Duration = 5 * time.Minute
	ConnectionIdleTimeoutDefault    time.Duration = 5 * time.Minute
	MetadataCacheTimeoutDefault     time.Duration = 5 * time.Minute
	MetadataCacheCleanupTimeDefault time.Duration = 5 * time.Minute
	BufferSizeMaxDefault            int64         = 1024 * 1024 * 64 // 64MB
)

// PathMapping ...
type IRODSFSPathMapping struct {
	IRODSPath      string `yaml:"irods_path" json:"irods_path"`
	MappingPath    string `yaml:"mapping_path" json:"mapping_path"`
	ResourceType   string `yaml:"resource_type" json:"resource_type"` // file or dir
	ReadOnly       bool   `yaml:"read_only" json:"read_only"`
	CreateDir      bool   `yaml:"create_dir" json:"create_dir"`
	IgnoreNotExist bool   `yaml:"ignore_not_exist" json:"ignore_not_exist"`
}

type IRODSFSConfig struct {
	Host         string               `yaml:"host"`
	Port         int                  `yaml:"port"`
	ProxyUser    string               `yaml:"proxy_user"`
	ClientUser   string               `yaml:"client_user"`
	Zone         string               `yaml:"zone"`
	Password     string               `yaml:"password"`
	PathMappings []IRODSFSPathMapping `yaml:"path_mappings"`
	UID          int                  `yaml:"uid"`
	GID          int                  `yaml:"gid"`
	SystemUser   string               `yaml:"system_user"`

	PoolHost string `yaml:"pool_host,omitempty"`
	PoolPort int    `yaml:"pool_port,omitempty"`

	ReadAheadMax             int           `yaml:"read_ahead_max"`
	OperationTimeout         time.Duration `yaml:"operation_timeout"`
	ConnectionIdleTimeout    time.Duration `yaml:"connection_idle_timeout"`
	ConnectionMax            int           `yaml:"connection_max"`
	MetadataCacheTimeout     time.Duration `yaml:"metadata_cache_timeout"`
	MetadataCacheCleanupTime time.Duration `yaml:"metadata_cache_cleanup_time"`
	BufferSizeMax            int64         `yaml:"buffer_size_max"`

	LogPath            string `yaml:"log_path,omitempty"`
	MonitorURL         string `yaml:"monitor_url,omitempty"`
	Profile            bool   `yaml:"profile,omitempty"`
	ProfileServicePort int    `yaml:"profile_service_port,omitempty"`
	AllowOther         bool   `yaml:"allow_other,omitempty"`
}

// NewDefaultIRODSFSConfig creates default IRODSFSConfig
func NewDefaultIRODSFSConfig() *IRODSFSConfig {
	return &IRODSFSConfig{
		Port:         PortDefault,
		PathMappings: []IRODSFSPathMapping{},
		UID:          -1,
		GID:          -1,
		SystemUser:   "",

		PoolHost: "",
		PoolPort: 0,

		ReadAheadMax:             ReadAheadMaxDefault,
		OperationTimeout:         OperationTimeoutDefault,
		ConnectionIdleTimeout:    ConnectionIdleTimeoutDefault,
		ConnectionMax:            ConnectionMaxDefault,
		MetadataCacheTimeout:     MetadataCacheTimeoutDefault,
		MetadataCacheCleanupTime: MetadataCacheCleanupTimeDefault,
		BufferSizeMax:            BufferSizeMaxDefault,

		LogPath:            "",
		MonitorURL:         "",
		Profile:            false,
		ProfileServicePort: 0,
		AllowOther:         true,
	}
}
