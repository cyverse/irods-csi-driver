package common

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/xerrors"
)

// KernelInfo stores kernel info
type KernelInfo struct {
	Major int
	Minor int
	Patch int
}

// GetKernelInfo returns kernel info
func GetKernelInfo() (*KernelInfo, error) {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return nil, xerrors.Errorf("failed to read os release file at %q", "/proc/sys/kernel/osrelease")
	}

	kernelVersion := strings.TrimSpace(string(data))
	kernelVersionSplits := strings.Split(kernelVersion, "-")

	if len(kernelVersionSplits) == 0 {
		return nil, xerrors.Errorf("failed to get kernel verion info out of string %q", kernelVersion)
	}

	kernelVersionParts := strings.Split(kernelVersionSplits[0], ".")
	if len(kernelVersionParts) != 3 {
		return nil, xerrors.Errorf("version string %q doesn't have 3 parts", kernelVersionSplits[0])
	}

	kv1, err := strconv.Atoi(kernelVersionParts[0])
	if err != nil {
		return nil, xerrors.Errorf("failed to convert string %q to int", kernelVersionParts[0])
	}

	kv2, err := strconv.Atoi(kernelVersionParts[1])
	if err != nil {
		return nil, xerrors.Errorf("failed to convert string %q to int", kernelVersionParts[1])
	}

	kv3, err := strconv.Atoi(kernelVersionParts[2])
	if err != nil {
		return nil, xerrors.Errorf("failed to convert string %q to int", kernelVersionParts[2])
	}

	return &KernelInfo{
		Major: kv1,
		Minor: kv2,
		Patch: kv3,
	}, nil
}

// HasHigherKernelVersionThan returns if the kernel version is higher or equal than given version
func (info *KernelInfo) HasHigherKernelVersionThan(major int, minor int, patch int) bool {
	if info.Major > major {
		return true
	}
	if info.Major < major {
		return false
	}
	// major is equal
	if info.Minor > minor {
		return true
	}
	if info.Minor < minor {
		return false
	}
	// minor is equal
	if info.Patch > patch {
		return true
	}
	if info.Patch < patch {
		return false
	}
	return true
}
