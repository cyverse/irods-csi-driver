/*
Following functions or objects are from the code under APL2 License.
- ParseEndpoint
Original code: https://github.com/kubernetes-sigs/aws-efs-csi-driver/blob/master/pkg/util/util.go


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

package common

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/xerrors"
)

// Config holds the parameters list which can be configured
type Config struct {
	Endpoint               string // CSI endpoint
	NodeID                 string // node ID
	SecretPath             string // Secret mount path
	PoolServiceEndpoint    string // iRODS FS Pool Service endpoint
	PrometheusExporterPort int    // Prometheus Exporter Service port
	StoragePath            string // Path to storage dir (for saving volume info and etc)
}

// NormalizeConfigKey normalizes config key
func NormalizeConfigKey(key string) string {
	key = strings.ToLower(key)

	if key == "driver" {
		return "client"
	}

	return strings.ReplaceAll(key, "_", "")
}

// MergeConfig merges configuration params
func MergeConfig(driverConfig *Config, driverSecrets map[string]string, volSecrets map[string]string, volParams map[string]string) map[string]string {
	configs := make(map[string]string)
	for k, v := range volSecrets {
		if len(v) > 0 {
			configs[NormalizeConfigKey(k)] = v
		}
	}

	for k, v := range volParams {
		if len(v) > 0 {
			configs[NormalizeConfigKey(k)] = v
		}
	}

	// driver secrets have higher priority
	for k, v := range driverSecrets {
		if len(v) > 0 {
			configs[NormalizeConfigKey(k)] = v
		}
	}

	if len(driverConfig.PoolServiceEndpoint) > 0 {
		configs[NormalizeConfigKey("pool_endpoint")] = driverConfig.PoolServiceEndpoint
	}

	if len(driverConfig.StoragePath) > 0 {
		configs[NormalizeConfigKey("storage_path")] = driverConfig.StoragePath
	}

	return configs
}

// RedactConfig redacts sensitive values
func RedactConfig(config map[string]string) map[string]string {
	newConfigs := make(map[string]string)
	for k, v := range config {
		if k == NormalizeConfigKey("password") || k == NormalizeConfigKey("irods_user_password") {
			newConfigs[k] = "**REDACTED**"
		} else {
			newConfigs[k] = v
		}
	}
	return newConfigs
}

// ParseEndpoint parses endpoint string (TCP or UNIX)
func ParseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", xerrors.Errorf("failed to parse endpoint %q: %w", endpoint, err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
	case "unix":
		addr = path.Join("/", addr)
		err := os.Remove(addr)
		if err != nil && !os.IsNotExist(err) {
			return "", "", xerrors.Errorf("failed to remove unix domain socket %q: %w", addr, err)
		}
	default:
		return "", "", xerrors.Errorf("unsupported protocol: %q", scheme)
	}

	return scheme, addr, nil
}

// ParsePoolServiceEndpoint parses endpoint string
func ParsePoolServiceEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", xerrors.Errorf("could not parse endpoint %q: %w", endpoint, err)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
		return u.Host, nil
	case "unix":
		path := path.Join("/", u.Path)
		return "unix://" + path, nil
	case "":
		if len(u.Host) > 0 {
			return u.Host, nil
		}
		return "", xerrors.Errorf("unknown host: %q", u.Host)
	default:
		return "", xerrors.Errorf("unsupported protocol: %q", scheme)
	}
}
