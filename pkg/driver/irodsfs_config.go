package driver

import "time"

// PathMapping ...
type IRODSFSPathMapping struct {
	IRODSPath    string `yaml:"irods_path"`
	MappingPath  string `yaml:"mapping_path"`
	ResourceType string `yaml:"resource_type"` // file or dir
}

type IRODSFSConfig struct {
	Host         string               `yaml:"host"`
	Port         int                  `yaml:"port"`
	ProxyUser    string               `yaml:"proxy_user,omitempty"`
	ClientUser   string               `yaml:"client_user"`
	Zone         string               `yaml:"zone"`
	Password     string               `yaml:"password,omitempty"`
	PathMappings []IRODSFSPathMapping `yaml:"path_mappings"`

	BlockSize             int           `yaml:"block_size"`
	ReadAheadMax          int           `yaml:"read_ahead_max"`
	UseBlockIO            bool          `yaml:"use_block_io"`
	PerFileBlockCacheMax  int           `yaml:"per_file_block_cache_max"`
	OperationTimeout      time.Duration `yaml:"operation_timeout"`
	ConnectionIdleTimeout time.Duration `yaml:"connection_idle_timeout"`
	ConnectionMax         int           `yaml:"connection_max"`
	CacheTimeout          time.Duration `yaml:"cache_timeout"`
	CacheCleanupTime      time.Duration `yaml:"cache_cleanup_time"`

	LogPath    string `yaml:"log_path,omitempty"`
	AllowOther bool   `yaml:"allow_other,omitempty"`
}
