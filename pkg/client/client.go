package client

import "strings"

// ClientType is a mount client type
type ClientType string

// mount driver (iRODS Client) types
const (
	// FuseClientType is for iRODS FUSE
	FuseClientType ClientType = "irodsfuse"
	// WebdavClientType is for WebDav client (Davfs2)
	WebdavClientType ClientType = "webdav"
	// NfsClientType is for NFS client
	NfsClientType ClientType = "nfs"
)

// GetClientType returns iRODS Client value from param map
func GetClientType(params map[string]string, secrets map[string]string, defaultClient ClientType) ClientType {
	irodsClient := ""
	for k, v := range secrets {
		if strings.ToLower(k) == "driver" || strings.ToLower(k) == "client" {
			irodsClient = v
			break
		}
	}

	for k, v := range params {
		if strings.ToLower(k) == "driver" || strings.ToLower(k) == "client" {
			irodsClient = v
			break
		}
	}

	return GetValidClientType(irodsClient, defaultClient)
}

// IsValidClientType checks if given client string is valid
func IsValidClientType(client string) bool {
	switch client {
	case string(FuseClientType):
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
func GetValidClientType(client string, defaultClient ClientType) ClientType {
	switch client {
	case string(FuseClientType):
		return FuseClientType
	case string(WebdavClientType):
		return WebdavClientType
	case string(NfsClientType):
		return NfsClientType
	default:
		return defaultClient
	}
}
