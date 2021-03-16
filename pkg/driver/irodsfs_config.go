package driver

import "time"

const (
	PortDefault                  int           = 1247
	PerFileBlockCacheMaxDefault  int           = 3
	ConnectionMaxDefault         int           = 10
	OperationTimeoutDefault      time.Duration = 5 * time.Minute
	ConnectionIdleTimeoutDefault time.Duration = 5 * time.Minute
	CacheTimeoutDefault          time.Duration = 5 * time.Minute
	CacheCleanupTimeDefault      time.Duration = 5 * time.Minute
)

// PathMapping ...
type IRODSFSPathMapping struct {
	IRODSPath    string `yaml:"irods_path" json:"irods_path"`
	MappingPath  string `yaml:"mapping_path" json:"mapping_path"`
	ResourceType string `yaml:"resource_type" json:"resource_type"` // file or dir
}

type IRODSFSConfig struct {
	Host         string               `yaml:"host"`
	Port         int                  `yaml:"port"`
	ProxyUser    string               `yaml:"proxy_user"`
	ClientUser   string               `yaml:"client_user"`
	Zone         string               `yaml:"zone"`
	Password     string               `yaml:"password"`
	PathMappings []IRODSFSPathMapping `yaml:"path_mappings"`

	PerFileBlockCacheMax  int           `yaml:"per_file_block_cache_max"`
	OperationTimeout      time.Duration `yaml:"operation_timeout"`
	ConnectionIdleTimeout time.Duration `yaml:"connection_idle_timeout"`
	ConnectionMax         int           `yaml:"connection_max"`
	CacheTimeout          time.Duration `yaml:"cache_timeout"`
	CacheCleanupTime      time.Duration `yaml:"cache_cleanup_time"`

	LogPath    string `yaml:"log_path,omitempty"`
	AllowOther bool   `yaml:"allow_other,omitempty"`
}

// NewDefaultIRODSFSConfig creates default IRODSFSConfig
func NewDefaultIRODSFSConfig() *IRODSFSConfig {
	return &IRODSFSConfig{
		Port:         PortDefault,
		PathMappings: []IRODSFSPathMapping{},

		PerFileBlockCacheMax:  PerFileBlockCacheMaxDefault,
		OperationTimeout:      OperationTimeoutDefault,
		ConnectionIdleTimeout: ConnectionIdleTimeoutDefault,
		ConnectionMax:         ConnectionMaxDefault,
		CacheTimeout:          CacheTimeoutDefault,
		CacheCleanupTime:      CacheCleanupTimeDefault,

		LogPath:    "",
		AllowOther: true,
	}
}
