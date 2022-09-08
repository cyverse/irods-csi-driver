package irods

import (
	"strconv"
	"strings"
)

// getConfigEnforceProxyAccess checks if proxy access is enforced via driver config
func getConfigEnforceProxyAccess(configs map[string]string) bool {
	for k, v := range configs {
		if strings.ToLower(k) == "enforceproxyaccess" {
			enforce, _ := strconv.ParseBool(v)
			return enforce
		}
	}
	return false
}

// getConfigMountPathWhitelist returns a whitelist of collections that users can mount
func getConfigMountPathWhitelist(configs map[string]string) []string {
	for k, v := range configs {
		if strings.ToLower(k) == "mountpathwhitelist" {
			whitelist := strings.Split(v, ",")
			for idx := range whitelist {
				whitelist[idx] = strings.TrimSpace(whitelist[idx])
			}

			return whitelist
		}
	}
	return []string{"/"}
}
