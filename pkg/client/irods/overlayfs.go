package irods

import (
	"strings"

	client_common "github.com/cyverse/irods-csi-driver/pkg/client/common"
	"k8s.io/klog"
)

// OverlayFSDriverType is a overlayfs driver type
type OverlayFSDriverType string

// overlayfs driver types
const (
	// OverlayDriverType is for kernel built-in overlay
	OverlayDriverType OverlayFSDriverType = "overlay"
	// FuseOverlayFSDriverType is for fuse-overlayfs
	FuseOverlayFSDriverType OverlayFSDriverType = "fuse-overlayfs"
)

// GetOverlayFSDriverType returns valid driver type
func GetOverlayFSDriverType(driver string) OverlayFSDriverType {
	switch strings.ToLower(driver) {
	case string(OverlayDriverType):
		return OverlayDriverType
	case string(FuseOverlayFSDriverType):
		return FuseOverlayFSDriverType
	default:
		return OverlayDriverType
	}
}

// IsOverlayDriverSupported returns if overlay driver is supported
func IsOverlayDriverSupported() bool {
	info, err := client_common.GetKernelInfo()
	if err != nil {
		klog.Errorf("failed to get kernel info %s", err)
		return false
	}

	// requires ubuntu 22 or similar
	// obviously, kernel 5.4.0 doesn't work
	return info.HasHigherKernelVersionThan(5, 15, 0)
}
