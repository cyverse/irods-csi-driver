package commons

import (
	"strings"

	"github.com/cockroachdb/errors"
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

// ParseClientType returns the requested client type. Omitting client preserves
// the existing iRODS FUSE default for backwards compatibility.
func ParseClientType(params map[string]string) (ClientType, error) {
	client := strings.ToLower(strings.TrimSpace(params[NormalizeConfigKey("client")]))
	if client == "" {
		return IrodsFuseClientType, nil
	}

	clientType := ClientType(client)
	if !IsValidClientType(clientType) {
		return "", errors.Errorf("unsupported client type %q", client)
	}
	return clientType, nil
}

// IsValidClientType checks whether client is supported.
func IsValidClientType(client ClientType) bool {
	switch client {
	case IrodsFuseClientType:
		return true
	case WebdavClientType:
		return true
	case NfsClientType:
		return true
	default:
		return false
	}
}
