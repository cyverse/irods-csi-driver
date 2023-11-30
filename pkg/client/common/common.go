package common

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// ClientType is a mount client type
type ClientType string

// mount driver (iRODS Client) types
const (
	// IrodsFuseClientType is for iRODS FUSE
	IrodsFuseClientType ClientType = "irodsfuse"
	// WebdavClientType is for WebDav client (Davfs2)
	WebdavClientType ClientType = "webdav"
	// NfsClientType is for NFS client
	NfsClientType ClientType = "nfs"
)

// GetClientType returns iRODS Client value from param map
func GetClientType(params map[string]string) ClientType {
	return GetValidClientType(params["client"])
}

// IsValidClientType checks if given client string is valid
func IsValidClientType(client string) bool {
	switch client {
	case string(IrodsFuseClientType):
		return true
	case string(WebdavClientType):
		return true
	case string(NfsClientType):
		return true
	default:
		return false
	}
}

// GetValidClientType checks if given client string is valid
func GetValidClientType(client string) ClientType {
	switch client {
	case string(IrodsFuseClientType):
		return IrodsFuseClientType
	case string(WebdavClientType):
		return WebdavClientType
	case string(NfsClientType):
		return NfsClientType
	default:
		return IrodsFuseClientType
	}
}

// GetConfigEnforceProxyAccess checks if proxy access is enforced via driver config
func GetConfigEnforceProxyAccess(configs map[string]string) bool {
	enforce := configs["enforceproxyaccess"]
	bEnforce, _ := strconv.ParseBool(enforce)
	return bEnforce
}

// GetConfigMountPathWhitelist returns a whitelist of collections that users can mount
func GetConfigMountPathWhitelist(configs map[string]string) []string {
	if whitelist, ok := configs["mountpathwhitelist"]; ok {
		if len(whitelist) > 0 {
			whitelistItems := strings.Split(whitelist, ",")
			for idx := range whitelistItems {
				item := strings.TrimSpace(whitelistItems[idx])
				if len(item) == 0 {
					item = "/"
				}

				whitelistItems[idx] = item
			}
			return whitelistItems
		}
	}

	return []string{"/"}
}

// GetConfigDataRootPath returns a data root path
func GetConfigDataRootPath(configs map[string]string, volID string) string {
	irodsClientType := GetClientType(configs)
	return filepath.Join(configs["storagepath"], string(irodsClientType), volID)
}

// GetConfigOverlayFSLowerPath returns a lower path for overlayfs
func GetConfigOverlayFSLowerPath(configs map[string]string, volID string) string {
	irodsClientType := GetClientType(configs)
	name := fmt.Sprintf("%s-overlayfs-lower", volID)
	return filepath.Join(configs["storagepath"], string(irodsClientType), name)
}

// GetConfigOverlayFSUpperPath returns a upper path for overlayfs
func GetConfigOverlayFSUpperPath(configs map[string]string, volID string) string {
	irodsClientType := GetClientType(configs)
	name := fmt.Sprintf("%s-overlayfs-upper", volID)
	return filepath.Join(configs["storagepath"], string(irodsClientType), name)
}

// GetConfigOverlayFSWorkDirPath returns a work dir path for overlayfs
func GetConfigOverlayFSWorkDirPath(configs map[string]string, volID string) string {
	irodsClientType := GetClientType(configs)
	name := fmt.Sprintf("%s-overlayfs-workdir", volID)
	return filepath.Join(configs["storagepath"], string(irodsClientType), name)
}
