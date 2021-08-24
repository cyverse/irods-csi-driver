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
	FileBufferStoragePathDefault    string        = "/tmp/irodsfs"
	FileBufferSizeMaxDefault        int64         = 1024 * 1024 * 1024 // 1GB
)

// PathMapping ...
type IRODSFSPathMapping struct {
	IRODSPath      string `yaml:"irods_path" json:"irods_path"`
	MappingPath    string `yaml:"mapping_path" json:"mapping_path"`
	ResourceType   string `yaml:"resource_type" json:"resource_type"` // file or dir
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

	ReadAheadMax             int           `yaml:"read_ahead_max"`
	OperationTimeout         time.Duration `yaml:"operation_timeout"`
	ConnectionIdleTimeout    time.Duration `yaml:"connection_idle_timeout"`
	ConnectionMax            int           `yaml:"connection_max"`
	MetadataCacheTimeout     time.Duration `yaml:"metadata_cache_timeout"`
	MetadataCacheCleanupTime time.Duration `yaml:"metadata_cache_cleanup_time"`
	FileBufferStoragePath    string        `yaml:"file_buffer_storage_path"`
	FileBufferSizeMax        int64         `yaml:"file_buffer_size_max"`

	LogPath    string `yaml:"log_path,omitempty"`
	MonitorURL string `yaml:"monitor_url,omitempty"`
	AllowOther bool   `yaml:"allow_other,omitempty"`
}

// NewDefaultIRODSFSConfig creates default IRODSFSConfig
func NewDefaultIRODSFSConfig() *IRODSFSConfig {
	return &IRODSFSConfig{
		Port:         PortDefault,
		PathMappings: []IRODSFSPathMapping{},
		UID:          -1,
		GID:          -1,
		SystemUser:   "",

		ReadAheadMax:             ReadAheadMaxDefault,
		OperationTimeout:         OperationTimeoutDefault,
		ConnectionIdleTimeout:    ConnectionIdleTimeoutDefault,
		ConnectionMax:            ConnectionMaxDefault,
		MetadataCacheTimeout:     MetadataCacheTimeoutDefault,
		MetadataCacheCleanupTime: MetadataCacheCleanupTimeDefault,
		FileBufferStoragePath:    FileBufferStoragePathDefault,
		FileBufferSizeMax:        FileBufferSizeMaxDefault,

		LogPath:    "",
		MonitorURL: "",
		AllowOther: true,
	}
}
