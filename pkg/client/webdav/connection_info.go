package webdav

import (
	"net/url"
	"strings"

	"github.com/cyverse/irods-csi-driver/pkg/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	webdavAnonymousUser string = "anonymous"
)

// WebDAVConnectionInfo class
type WebDAVConnectionInfo struct {
	URL      string
	User     string
	Password string
	Config   map[string]string
}

// SetAnonymousUser sets anonymous user
func (connInfo *WebDAVConnectionInfo) SetAnonymousUser() {
	connInfo.User = webdavAnonymousUser
}

// IsAnonymousUser checks if the user is anonymous
func (connInfo *WebDAVConnectionInfo) IsAnonymousUser() bool {
	return connInfo.User == webdavAnonymousUser
}

func getConnectionInfoFromMap(params map[string]string, connInfo *WebDAVConnectionInfo) error {
	for k, v := range params {
		switch common.NormalizeConfigKey(k) {
		case common.NormalizeConfigKey("user"), common.NormalizeConfigKey("username"):
			connInfo.User = v
		case common.NormalizeConfigKey("password"), common.NormalizeConfigKey("user_password"):
			connInfo.Password = v
		case common.NormalizeConfigKey("url"):
			connInfo.URL = v
		case common.NormalizeConfigKey("config"):
			connInfo.Config = map[string]string{}
			configStrings := strings.Split(v, ",")
			for _, configString := range configStrings {
				configKV := strings.Split(strings.TrimSpace(configString), "=")
				if len(configKV) == 2 {
					connInfo.Config[strings.TrimSpace(configKV[0])] = strings.TrimSpace(configKV[1])
				}
			}
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo extracts WebDAVConnectionInfo value from param map
func GetConnectionInfo(configs map[string]string) (*WebDAVConnectionInfo, error) {
	connInfo := WebDAVConnectionInfo{}

	err := getConnectionInfoFromMap(configs, &connInfo)
	if err != nil {
		return nil, err
	}

	// user and password fields are optional
	// if user is not given, it is regarded as anonymous user
	if len(connInfo.User) == 0 {
		connInfo.SetAnonymousUser()
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && !connInfo.IsAnonymousUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(connInfo.URL) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument url is empty")
	}

	// check
	_, err = url.ParseRequestURI(connInfo.URL)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid URL - %q", connInfo.URL)
	}

	return &connInfo, nil
}
