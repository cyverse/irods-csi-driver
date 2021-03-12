package driver

import (
	"path/filepath"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClientType is a mount client type
type ClientType string

// mount driver (iRODS Client) types
const (
	// FuseType is for iRODS FUSE
	FuseType ClientType = "irodsfuse"
	// WebdavType is for WebDav client (Davfs2)
	WebdavType ClientType = "webdav"
	// NfsType is for NFS client
	NfsType ClientType = "nfs"
)

// IRODSConnectionInfo class
type IRODSConnectionInfo struct {
	Hostname   string
	Port       int
	Zone       string
	User       string
	Password   string
	ClientUser string // if this field has a value, user and password fields have proxy user info
	Path       string
}

// IRODSWebDAVConnectionInfo class
type IRODSWebDAVConnectionInfo struct {
	URL      string
	User     string
	Password string
}

// IRODSNFSConnectionInfo class
type IRODSNFSConnectionInfo struct {
	Hostname string
	Port     int
	Path     string
}

// NewIRODSConnectionInfo returns a new instance of IRODSConnectionInfo
func NewIRODSConnectionInfo(hostname string, port int, zone string, user string, password string, clientUser string, path string) *IRODSConnectionInfo {
	return &IRODSConnectionInfo{
		Hostname:   hostname,
		Port:       port,
		Zone:       zone,
		User:       user,
		Password:   password,
		ClientUser: clientUser,
		Path:       path,
	}
}

// NewIRODSWebDAVConnectionInfo returns a new instance of IRODSWebDAVConnectionInfo
func NewIRODSWebDAVConnectionInfo(url string, user string, password string) *IRODSWebDAVConnectionInfo {
	return &IRODSWebDAVConnectionInfo{
		URL:      url,
		User:     user,
		Password: password,
	}
}

// NewIRODSNFSConnectionInfo returns a new instance of IRODSNFSConnectionInfo
func NewIRODSNFSConnectionInfo(hostname string, port int, path string) *IRODSNFSConnectionInfo {
	return &IRODSNFSConnectionInfo{
		Hostname: hostname,
		Port:     port,
		Path:     path,
	}
}

// ExtractIRODSClientType extracts iRODS Client value from param map
func ExtractIRODSClientType(params map[string]string, secrets map[string]string, defaultClient ClientType) ClientType {
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

	return GetValidiRODSClientType(irodsClient, defaultClient)
}

// IsValidIRODSClientType checks if given client string is valid
func IsValidIRODSClientType(client string) bool {
	switch client {
	case string(FuseType):
		return true
	case string(WebdavType):
		return true
	case string(NfsType):
		return true
	default:
		return false
	}
}

// GetValidiRODSClientType checks if given client string is valid
func GetValidiRODSClientType(client string, defaultClient ClientType) ClientType {
	switch client {
	case string(FuseType):
		return FuseType
	case string(WebdavType):
		return WebdavType
	case string(NfsType):
		return NfsType
	default:
		return defaultClient
	}
}

// ExtractIRODSConnectionInfo extracts IRODSConnectionInfo value from param map
func ExtractIRODSConnectionInfo(params map[string]string, secrets map[string]string) (*IRODSConnectionInfo, error) {
	var user, password, clientUser, host, zone, path string
	port := 0

	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "user":
			user = v
		case "password":
			password = v
		case "clientuser":
			// for proxy
			clientUser = v
		case "host":
			host = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			port = p
		case "zone":
			zone = v
		case "path":
			if !filepath.IsAbs(v) {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}
			path = v
		default:
			// ignore
		}
	}

	for k, v := range params {
		switch strings.ToLower(k) {
		case "user":
			user = v
		case "password":
			password = v
		case "clientuser":
			// for proxy
			clientUser = v
		case "host":
			host = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			port = p
		case "zone":
			zone = v
		case "path":
			if !filepath.IsAbs(v) {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}
			path = v
		default:
			// ignore
		}
	}

	if len(user) == 0 {
		user = "anonymous"
	}

	// password can be empty for anonymous access
	if len(password) == 0 && user != "anonymous" {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(host) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host is empty")
	}

	if len(zone) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument zone is empty")
	}

	// port is optional
	if port <= 0 {
		// default
		port = 1247
	}

	conn := NewIRODSConnectionInfo(host, port, zone, user, password, clientUser, path)
	return conn, nil
}

// ExtractIRODSWebDAVConnectionInfo extracts IRODSWebDAVConnectionInfo value from param map
func ExtractIRODSWebDAVConnectionInfo(params map[string]string, secrets map[string]string) (*IRODSWebDAVConnectionInfo, error) {
	var user, password, url string

	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "user":
			user = v
		case "password":
			password = v
		case "url":
			url = v
		default:
			// ignore
		}
	}

	for k, v := range params {
		switch strings.ToLower(k) {
		case "user":
			user = v
		case "password":
			password = v
		case "url":
			url = v
		default:
			// ignore
		}
	}

	// user and password fields are optional
	// if user is not given, it is regarded as anonymous user
	if len(user) == 0 {
		user = "anonymous"
	}

	// password can be empty for anonymous access
	if len(password) == 0 && user != "anonymous" {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(url) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument url is empty")
	}

	conn := NewIRODSWebDAVConnectionInfo(url, user, password)
	return conn, nil
}

// ExtractIRODSNFSConnectionInfo extracts IRODSNFSConnectionInfo value from param map
func ExtractIRODSNFSConnectionInfo(params map[string]string, secrets map[string]string) (*IRODSNFSConnectionInfo, error) {
	var host, path string
	port := 0

	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "host":
			host = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			port = p
		case "path":
			path = v
		default:
			// ignore
		}
	}

	for k, v := range params {
		switch strings.ToLower(k) {
		case "host":
			host = v
		case "port":
			p, err := strconv.Atoi(v)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Argument %q must be a valid port number - %s", k, err)
			}
			port = p
		case "path":
			path = v
		default:
			// ignore
		}
	}

	if len(host) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host is empty")
	}

	if len(path) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument path is empty")
	}

	if port <= 0 {
		// default
		port = 2049
	}

	conn := NewIRODSNFSConnectionInfo(host, port, path)
	return conn, nil
}
