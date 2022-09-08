package webdav

import (
	"net/url"

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
		switch k {
		case "user":
			connInfo.User = v
		case "password":
			connInfo.Password = v
		case "url":
			connInfo.URL = v
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
		return nil, status.Errorf(codes.InvalidArgument, "Invalid URL - %s", connInfo.URL)
	}

	return &connInfo, nil
}
