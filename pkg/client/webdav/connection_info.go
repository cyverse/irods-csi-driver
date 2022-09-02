package webdav

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WebDAVConnectionInfo class
type WebDAVConnectionInfo struct {
	URL      string
	User     string
	Password string
}

func getConnectionInfoFromMap(params map[string]string, connInfo *WebDAVConnectionInfo) error {
	for k, v := range params {
		switch strings.ToLower(k) {
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
func GetConnectionInfo(params map[string]string, secrets map[string]string) (*WebDAVConnectionInfo, error) {
	connInfo := WebDAVConnectionInfo{}

	err := getConnectionInfoFromMap(secrets, &connInfo)
	if err != nil {
		return nil, err
	}

	err = getConnectionInfoFromMap(params, &connInfo)
	if err != nil {
		return nil, err
	}

	// user and password fields are optional
	// if user is not given, it is regarded as anonymous user
	if len(connInfo.User) == 0 {
		connInfo.User = "anonymous"
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && connInfo.User != "anonymous" {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(connInfo.URL) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument url is empty")
	}

	return &connInfo, nil
}
