package driver

import (
	"fmt"
	"os"
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
		return nil, status.Errorf(codes.Internal, "Secret path %q does not exist - %v", secretPath, err)
	}

	if !exist {
		return nil, status.Errorf(codes.NotFound, "Secret path %q does not exist", secretPath)
	}

	secrets := make(map[string]string)

	files, err := os.ReadDir(secretPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			k := path.Base(file.Name())
			fullPath := path.Join(secretPath, k)
			content, readErr := os.ReadFile(fullPath)
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
		if k == common.NormalizeConfigKey("provisioning_mode") {
			if strings.ToLower(v) == "dynamic" {
				return true
			}
		}
	}

	return false
}

func setDynamicVolumeProvisioningMode(volContext map[string]string) {
	volContext[common.NormalizeConfigKey("provisioning_mode")] = "dynamic"
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
		switch common.NormalizeConfigKey(k) {
		case common.NormalizeConfigKey("volume_root_path"):
			if !filepath.IsAbs(v) {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be an absolute path", k)
			}
			if v == "/" {
				config.VolumeRootPath = v
			} else {
				config.VolumeRootPath = strings.TrimRight(v, "/")
			}
		case common.NormalizeConfigKey("retain_data"):
			retain, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %v", k, err)
			}
			config.RetainData = retain
		case common.NormalizeConfigKey("no_volume_dir"):
			novolumedir, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Argument %q must be a boolean value - %v", k, err)
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
