package driver

import (
	"encoding/base64"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	dynamicProvisioningMode string = "dynamic"
	volumeIDPrefix          string = "volid-"
)

var (
	volumeCaps = []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	}
)

// isValidVolumeCapabilities checks validity of volume capabilities
func isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, m := range volumeCaps {
			if m == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
		}
	}
	return foundAll
}

func isDynamicVolumeProvisioningMode(volumeContext map[string]string) bool {
	for key, value := range volumeContext {
		if key == "provisioningMode" {
			return strings.EqualFold(strings.TrimSpace(value), dynamicProvisioningMode)
		}
	}

	return false
}

func setDynamicVolumeProvisioningMode(volumeContext map[string]string) {
	volumeContext["provisioningMode"] = dynamicProvisioningMode
}

type controllerConfig struct {
	volumeRootPath        string
	volumePath            string
	createVolumeDirectory bool
}

func parseControllerConfig(volumeName string, params map[string]string) (*controllerConfig, error) {
	if err := validateVolumeName(volumeName); err != nil {
		return nil, err
	}

	config := &controllerConfig{createVolumeDirectory: true}
	for key, value := range params {
		switch key {
		case "volumeRootPath":
			if !path.IsAbs(value) {
				return nil, status.Errorf(codes.InvalidArgument, "parameter %q must be an absolute path", key)
			}
			config.volumeRootPath = path.Clean(value)
		case "noVolumeDir":
			noVolumeDirectory, err := strconv.ParseBool(value)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "parameter %q must be a boolean: %v", key, err)
			}
			config.createVolumeDirectory = !noVolumeDirectory
		}
	}

	if config.volumeRootPath == "" {
		return nil, status.Error(codes.InvalidArgument, "parameter \"volumeRootPath\" is required")
	}

	if !config.createVolumeDirectory {
		config.volumePath = config.volumeRootPath
	} else {
		config.volumePath = path.Join(config.volumeRootPath, volumeName)
	}

	return config, nil
}

// generateVolumeID deterministically derives a reversible, CSI-safe volume ID
// from a CSI volume name.
func generateVolumeID(volName string) string {
	return volumeIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(volName))
}

// volumeNameFromID returns the CSI volume name encoded in a driver volume ID.
func volumeNameFromID(volumeID string) (string, error) {
	encodedName, found := strings.CutPrefix(volumeID, volumeIDPrefix)
	if !found || encodedName == "" {
		return "", fmt.Errorf("invalid driver volume ID %q", volumeID)
	}

	nameBytes, err := base64.RawURLEncoding.DecodeString(encodedName)
	if err != nil {
		return "", fmt.Errorf("decode driver volume ID %q: %w", volumeID, err)
	}
	volumeName := string(nameBytes)
	if err := validateVolumeName(volumeName); err != nil {
		return "", err
	}

	return volumeName, nil
}

func validateVolumeName(volumeName string) error {
	if volumeName == "" || volumeName == "." || volumeName == ".." || path.Base(volumeName) != volumeName {
		return status.Errorf(codes.InvalidArgument, "volume name %q must be a single path segment", volumeName)
	}

	return nil
}
