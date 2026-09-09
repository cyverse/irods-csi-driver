package commons

import (
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

// Config holds the parameters list which can be configured
type Config struct {
	ServiceEndpoint         string // CSI Service endpoint
	NodeID                  string // node ID
	SecretPath              string // Secret mount path
	IRODSFSDServiceEndpoint string // iRODS FSD Service endpoint
	PrometheusExporterPort  int    // Prometheus Exporter Service port
}

func (config *Config) GetServiceEndpoint() string {
	if len(config.ServiceEndpoint) > 0 {
		return config.ServiceEndpoint
	}

	return "unix:///tmp/csi.sock"
}

func (config *Config) GetIRODSFSDServiceEndpoint() string {
	if len(config.IRODSFSDServiceEndpoint) > 0 {
		return config.IRODSFSDServiceEndpoint
	}

	return "tcp://127.0.0.1:13020"
}

// MakeWorkDirs makes dirs required
func (config *Config) MakeWorkDirs() error {
	scheme, endpoint, err := ParseServiceEndpoint(config.GetServiceEndpoint())
	if err != nil {
		return err
	}

	if scheme == "unix" {
		err = config.makeUnixSocketDir(endpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

// makeUnixSocketDir makes unix socket dir
func (config *Config) makeUnixSocketDir(endpoint string) error {
	parentDir := filepath.Dir(endpoint)
	unixSocketDirInfo, err := os.Stat(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			err2 := os.MkdirAll(parentDir, os.FileMode(0777))
			if err2 != nil {
				return errors.Wrapf(err2, "failed to make a directory for unix socket %q", parentDir)
			}
			return nil
		} else {
			return errors.Wrapf(err, "unix socket directory %q error", parentDir)
		}
	} else if !unixSocketDirInfo.IsDir() {
		return errors.Newf("unix socket parent path %q is not a directory", parentDir)
	} else {
		unixSocketDirPerm := unixSocketDirInfo.Mode().Perm()
		if unixSocketDirPerm&0200 != 0200 {
			return errors.Newf("unix socket directory %q must have write permission", parentDir)
		}
		// ok - fall
	}

	// endpoint is a file
	fileInfo, err := os.Lstat(endpoint)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "service unix socket file %q error", endpoint)
		}
	} else {
		if fileInfo.Mode()&os.ModeSocket == 0 {
			return errors.Newf("service endpoint %q exists but is not a unix socket", endpoint)
		}

		err2 := os.Remove(endpoint)
		if err2 != nil {
			return errors.Wrapf(err2, "failed to remove the existing unix socket file %q", endpoint)
		}
	}

	return nil
}

// Validate validates configuration
func (config *Config) Validate() error {
	_, _, err := ParseServiceEndpoint(config.GetServiceEndpoint())
	if err != nil {
		return err
	}

	if len(config.NodeID) == 0 {
		return errors.Errorf("node ID must be given")
	}

	if len(config.SecretPath) > 0 {
		if !filepath.IsAbs(config.SecretPath) {
			return errors.Newf("secret path must be an absolute path %q", config.SecretPath)
		}
	}

	_, _, err = ParseServiceEndpoint(config.GetIRODSFSDServiceEndpoint())
	if err != nil {
		return err
	}

	if config.PrometheusExporterPort < 0 || config.PrometheusExporterPort > 65535 {
		return errors.New("prometheus_exporter_port must be between 0 and 65535")
	}

	return nil
}

// MergeConfig merges configuration params
func MergeConfig(driverConfig *Config, driverSecrets map[string]string, volSecrets map[string]string, volParams map[string]string) map[string]string {
	configs := make(map[string]string)
	for k, v := range volSecrets {
		if len(v) > 0 {
			configs[k] = v
		}
	}

	for k, v := range volParams {
		if len(v) > 0 {
			configs[k] = v
		}
	}

	// driver secrets have higher priority
	for k, v := range driverSecrets {
		if len(v) > 0 {
			configs[k] = v
		}
	}

	return configs
}

// RedactConfig redacts sensitive values
func RedactConfig(config map[string]string) map[string]string {
	newConfigs := make(map[string]string)
	for k, v := range config {
		if k == "password" {
			newConfigs[k] = "**REDACTED**"
		} else {
			newConfigs[k] = v
		}
	}
	return newConfigs
}
