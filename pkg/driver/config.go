/*
Copyright 2019 The Kubernetes Authors.
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
    "net/url"
    "os"
    "path"
    "path/filepath"
    "strings"

	"github.com/pkg/errors"
)

const (
    FuseType = "fuse"
    NfsType = "nfs"
    WebdavType = "webdav"
)

// hold the parameters list which can be configured
type Config struct {
	DriverType string // driver type [fuse|nfs|webdav]
	Endpoint string // CSI endpoint
    NodeID string // node ID
}

// validate the driver type
func ValidateDriverType(driverType string) error {
	if driverType == "" {
		return errors.New("driver type is empty")
	}

    switch(driverType) {
    case FuseType:
        fallthrough
    case NfsType:
        fallthrough
    case WebdavType:
        return nil

    default:
        return fmt.Errorf("unknown driver type - %v", driverType)
    }
}

// parse endpoint (TCP or UNIX)
func ParseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("could not parse endpoint: %v", err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
	case "unix":
		addr = path.Join("/", addr)
        err := os.Remove(addr)
		if err != nil && !os.IsNotExist(err) {
			return "", "", fmt.Errorf("could not remove unix domain socket %q: %v", addr, err)
		}
	default:
		return "", "", fmt.Errorf("unsupported protocol: %s", scheme)
	}

	return scheme, addr, nil
}
