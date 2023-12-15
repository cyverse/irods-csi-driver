package irods

import (
	"time"

	irodsfs_common_vpath "github.com/cyverse/irodsfs-common/vpath"
)

// this structures must match to iRODS FUSE Lite Config
// https://github.com/cyverse/irodsfs/blob/main/commons/config.go#L80

const (
	PortDefault                     int           = 1247
	ReadAheadMaxDefault             int           = 1024 * 64 // 64KB
	ConnectionMaxDefault            int           = 10
	OperationTimeoutDefault         time.Duration = 5 * time.Minute
	ConnectionLifespanDefault       time.Duration = 1 * time.Hour
	ConnectionIdleTimeoutDefault    time.Duration = 5 * time.Minute
	MetadataCacheTimeoutDefault     time.Duration = 5 * time.Minute
	MetadataCacheCleanupTimeDefault time.Duration = 5 * time.Minute
)

type IRODSFSConfig struct {
	Host              string                              `yaml:"host"`
	Port              int                                 `yaml:"port"`
	ProxyUser         string                              `yaml:"proxy_user"`
	ClientUser        string                              `yaml:"client_user"`
	Zone              string                              `yaml:"zone"`
	Password          string                              `yaml:"password"`
	Resource          string                              `yaml:"resource,omitempty"`
	PathMappings      []irodsfs_common_vpath.VPathMapping `yaml:"path_mappings"`
	NoPermissionCheck bool                                `yaml:"no_permission_check"`
	NoSetXattr        bool                                `yaml:"no_set_xattr"`
	UID               int                                 `yaml:"uid"`
	GID               int                                 `yaml:"gid"`
	SystemUser        string                              `yaml:"system_user"`

	DataRootPath string `yaml:"data_root_path,omitempty"`

	LogPath string `yaml:"log_path,omitempty"`

	PoolEndpoint string `yaml:"pool_endpoint,omitempty"`

	ReadAheadMax             int           `yaml:"read_ahead_max"`
	OperationTimeout         time.Duration `yaml:"operation_timeout"`
	ConnectionLifespan       time.Duration `yaml:"connection_lifespan"`
	ConnectionIdleTimeout    time.Duration `yaml:"connection_idle_timeout"`
	ConnectionMax            int           `yaml:"connection_max"`
	MetadataCacheTimeout     time.Duration `yaml:"metadata_cache_timeout"`
	MetadataCacheCleanupTime time.Duration `yaml:"metadata_cache_cleanup_time"`

	MonitorURL string `yaml:"monitor_url,omitempty"`

	Profile            bool `yaml:"profile,omitempty"`
	ProfileServicePort int  `yaml:"profile_service_port,omitempty"`

	AllowOther bool   `yaml:"allow_other,omitempty"`
	InstanceID string `yaml:"instanceid,omitempty"`
}

// NewDefaultIRODSFSConfig creates default IRODSFSConfig
func NewDefaultIRODSFSConfig() *IRODSFSConfig {
	return &IRODSFSConfig{
		Host:              "",
		Port:              PortDefault,
		ProxyUser:         "",
		ClientUser:        "",
		Zone:              "",
		Password:          "",
		Resource:          "",
		PathMappings:      []irodsfs_common_vpath.VPathMapping{},
		NoPermissionCheck: false,
		NoSetXattr:        false,
		UID:               -1,
		GID:               -1,
		SystemUser:        "",

		DataRootPath: "/storage",

		LogPath: "", // use default

		PoolEndpoint: "",

		ReadAheadMax:             ReadAheadMaxDefault,
		OperationTimeout:         OperationTimeoutDefault,
		ConnectionLifespan:       ConnectionLifespanDefault,
		ConnectionIdleTimeout:    ConnectionIdleTimeoutDefault,
		ConnectionMax:            ConnectionMaxDefault,
		MetadataCacheTimeout:     MetadataCacheTimeoutDefault,
		MetadataCacheCleanupTime: MetadataCacheCleanupTimeDefault,

		MonitorURL: "",

		Profile:            false,
		ProfileServicePort: 0,

		AllowOther: true,
		InstanceID: "",
	}
}
