/*
Copyright 2020 CyVerse
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mount driver (iRODS Client) types
const (
	// FuseType is for iRODS FUSE
	FuseType = "irodsfuse"
	// WebdavType is for WebDav client (Davfs2)
	WebdavType = "webdav"
	// NfsType is for NFS client
	NfsType = "nfs"
)

// IRODSConnection class
type IRODSConnection struct {
	Hostname   string
	Port       int
	Zone       string
	User       string
	Password   string
	ClientUser string // if this field has a value, user and password fields have proxy user info
	Path       string
}

// IRODSWebDAVConnection class
type IRODSWebDAVConnection struct {
	URL      string
	User     string
	Password string
}

// IRODSNFSConnection class
type IRODSNFSConnection struct {
	Hostname string
	Port     int
	Path     string
}

// NewIRODSConnection returns a new instance of IRODSConnection
func NewIRODSConnection(hostname string, port int, zone string, user string, password string, clientUser string, path string) *IRODSConnection {
	return &IRODSConnection{
		Hostname:   hostname,
		Port:       port,
		Zone:       zone,
		User:       user,
		Password:   password,
		ClientUser: clientUser,
		Path:       path,
	}
}

// GetHostArgs returns host arguments
func (conn *IRODSConnection) GetHostArgs() []string {
	hostport := fmt.Sprintf("%s:%d", conn.Hostname, conn.Port)
	args := []string{hostport, conn.Zone}
	return args
}

// GetLoginInfoArgs returns login arguments
func (conn *IRODSConnection) GetLoginInfoArgs() []string {
	stdinValues := []string{conn.User, conn.Password, conn.ClientUser}
	return stdinValues
}

// NewIRODSWebDAVConnection returns a new instance of IRODSWebDAVConnection
func NewIRODSWebDAVConnection(url string, user string, password string) *IRODSWebDAVConnection {
	return &IRODSWebDAVConnection{
		URL:      url,
		User:     user,
		Password: password,
	}
}

// NewIRODSNFSConnection returns a new instance of IRODSNFSConnection
func NewIRODSNFSConnection(hostname string, port int, path string) *IRODSNFSConnection {
	return &IRODSNFSConnection{
		Hostname: hostname,
		Port:     port,
		Path:     path,
	}
}

// ExtractIRODSClientType extracts iRODS Client value from param map
func ExtractIRODSClientType(params map[string]string, secrets map[string]string, defaultClient string) string {
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

	if IsValidIRODSClientType(irodsClient) {
		return irodsClient
	}

	return defaultClient
}

// IsValidIRODSClientType checks if given client string is valid
func IsValidIRODSClientType(client string) bool {
	switch client {
	case FuseType:
		return true
	case WebdavType:
		return true
	case NfsType:
		return true
	default:
		return false
	}
}

// ExtractIRODSConnection extracts IRODSConnection value from param map
func ExtractIRODSConnection(params map[string]string, secrets map[string]string) (*IRODSConnection, error) {
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
		return nil, status.Error(codes.InvalidArgument, "Argument user is empty")
	}

	if len(password) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(host) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument host is empty")
	}

	if len(zone) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument zone is empty")
	}

	// path is optional

	if port <= 0 {
		// default
		port = 1247
	}

	conn := NewIRODSConnection(host, port, zone, user, password, clientUser, path)
	return conn, nil
}

// ExtractIRODSWebDAVConnection extracts IRODSWebDAVConnection value from param map
func ExtractIRODSWebDAVConnection(params map[string]string, secrets map[string]string) (*IRODSWebDAVConnection, error) {
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

	if len(url) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument url is empty")
	}

	conn := NewIRODSWebDAVConnection(url, user, password)
	return conn, nil
}

// ExtractIRODSNFSConnection extracts IRODSNFSConnection value from param map
func ExtractIRODSNFSConnection(params map[string]string, secrets map[string]string) (*IRODSNFSConnection, error) {
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

	conn := NewIRODSNFSConnection(host, port, path)
	return conn, nil
}
