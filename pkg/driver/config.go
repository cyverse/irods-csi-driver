package driver

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cyverse/irods-csi-driver/pkg/common"
	"github.com/cyverse/irods-csi-driver/pkg/mounter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// readSecrets reads secrets from secret volume mount
func readSecrets(secretPath string) (map[string]string, error) {
	exist, err := mounter.PathExists(secretPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Secret path %s does not exist - %s", secretPath, err)
	}

	if !exist {
		return nil, status.Errorf(codes.NotFound, "Secret path %s does not exist", secretPath)
	}

	secrets := make(map[string]string)

	files, err := ioutil.ReadDir(secretPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			k := path.Base(file.Name())
			fullPath := path.Join(secretPath, k)
			content, readErr := ioutil.ReadFile(fullPath)
			if readErr == nil {
				contentString := string(content)
				v := strings.TrimSpace(contentString)
				secrets[k] = v
			}
		}
	}
	return secrets, nil
}

func isDynamicVolumeProvisioningMode(volContext map[string]string) bool {
	for k, v := range volContext {
		if k == "provisioningmode" {
			if strings.ToLower(v) == "dynamic" {
				return true
			}
		}
	}

	return false
}

func setDynamicVolumeProvisioningMode(volContext map[string]string) {
	volContext["provisioningmode"] = "dynamic"
}

// ControllerConfig is a controller config struct
type ControllerConfig struct {
	VolumeRootPath     string
	VolumePath         string
	RetainData         bool
	NotCreateVolumeDir bool
}

func getControllerConfigFromMap(params map[string]string, config *ControllerConfig) error {
	for k, v := range params {
		switch k {
		case "volumerootpath":
			if !filepath.IsAbs(v) {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}
			if v == "/" {
				config.VolumeRootPath = v
			} else {
				config.VolumeRootPath = strings.TrimRight(v, "/")
			}
		case "retaindata":
			retain, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			config.RetainData = retain
		case "novolumedir":
			novolumedir, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %s", k, err)
			}
			config.NotCreateVolumeDir = novolumedir
		default:
			// ignore
		}
	}

	return nil
}

// makeControllerConfig extracts ControllerConfig value from param map
func makeControllerConfig(volName string, configs map[string]string) (*ControllerConfig, error) {
	controllerConfig := ControllerConfig{
		VolumeRootPath:     "",
		VolumePath:         "",
		RetainData:         false,
		NotCreateVolumeDir: false,
	}

	err := getControllerConfigFromMap(configs, &controllerConfig)
	if err != nil {
		return nil, err
	}

	if len(controllerConfig.VolumeRootPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument volumeRootPath is not provided")
	}

	if controllerConfig.NotCreateVolumeDir {
		controllerConfig.VolumePath = controllerConfig.VolumeRootPath
		// in this case, we should retain data because the mounted path may have files
		// we should not delete these old files when the pvc is deleted
		controllerConfig.RetainData = true
	} else {
		controllerConfig.VolumePath = fmt.Sprintf("%s/%s", controllerConfig.VolumeRootPath, volName)
	}

	return &controllerConfig, nil
}

func normalizeConfigKey(key string) string {
	key = strings.ToLower(key)

	if key == "driver" {
		return "client"
	}

	return strings.ReplaceAll(key, "_", "")
}

// mergeConfig merges configuration params
func mergeConfig(driverConfig *common.Config, driverSecrets map[string]string, volSecrets map[string]string, volParams map[string]string) map[string]string {
	configs := make(map[string]string)
	for k, v := range volSecrets {
		if len(v) > 0 {
			configs[k] = normalizeConfigKey(v)
		}
	}

	for k, v := range volParams {
		if len(v) > 0 {
			configs[k] = normalizeConfigKey(v)
		}
	}

	// driver secrets have higher priority
	for k, v := range driverSecrets {
		if len(v) > 0 {
			configs[k] = normalizeConfigKey(v)
		}
	}

	if len(driverConfig.PoolServiceEndpoint) > 0 {
		configs["poolendpoint"] = driverConfig.PoolServiceEndpoint
	}

	return configs
}
