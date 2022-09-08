package driver

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/rs/xid"
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

// generateVolumeID generates volume id from volume name
func generateVolumeID(volName string) string {
	return fmt.Sprintf("volid-%s-%s", volName, xid.New().String())
}
