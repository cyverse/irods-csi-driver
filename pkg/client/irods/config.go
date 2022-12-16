package irods

import (
	"path/filepath"
	"strconv"
	"strings"
)

// getConfigEnforceProxyAccess checks if proxy access is enforced via driver config
func getConfigEnforceProxyAccess(configs map[string]string) bool {
	enforce := configs["enforceproxyaccess"]
	bEnforce, _ := strconv.ParseBool(enforce)
	return bEnforce
}

// getConfigMountPathWhitelist returns a whitelist of collections that users can mount
func getConfigMountPathWhitelist(configs map[string]string) []string {
	whitelist := configs["mountpathwhitelist"]

	whitelistItems := strings.Split(whitelist, ",")
	if len(whitelistItems) > 0 {
		for idx := range whitelistItems {
			whitelistItems[idx] = strings.TrimSpace(whitelistItems[idx])
		}
		return whitelistItems
	}

	return []string{"/"}
}

func getConfigDataRootPath(configs map[string]string, volID string) string {
	return filepath.Join(configs["storagepath"], "irodsfs", volID)
}
