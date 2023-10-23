/*
Following functions or objects are from the code under APL2 License.
- Mounter
- NodeMounter
- newNodeMounter
- GetDeviceName
- Mount
- MountSensitive
- doMount
- MakeMountArgsSensitive
- sanitizedOptionsForLogging
- Unmount
- List
- ListProcMounts
- parseProcMounts
- IsLikelyNotMountPoint
- GetMountRefs
- SearchMountPoints
- PathExists
- MakeDir
Original code:
- https://github.com/kubernetes-sigs/aws-efs-csi-driver/blob/master/pkg/driver/mounter.go
- https://github.com/kubernetes/utils/blob/master/mount/mount.go


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

package mounter

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/xerrors"
	"k8s.io/klog/v2"
	utilio "k8s.io/utils/io"
	"k8s.io/utils/mount"
)

const (
	// Default mount command if mounter path is not specified.
	defaultMountCommand = "mount"
	// Log message where sensitive mount options were removed
	sensitiveOptionsRemoved = "<masked>"
	// Number of fields per line in /proc/mounts as per the fstab man page.
	expectedNumFieldsPerLine = 6
	// Location of the mount file to use
	procMountsPath = "/proc/mounts"
	// Location of the mountinfo file
	procMountInfoPath = "/proc/self/mountinfo"
)

type Mounter interface {
	mount.Interface
	GetDeviceName(mountPath string) (string, int, error)
	MountSensitive2(source string, sourceMasked string, target string, fstype string, options []string, sensitiveOptions []string, stdinValues []string) error
	FuseUnmount(target string, lazy bool) error
}

type NodeMounter struct {
	mount.Interface
}

func NewNodeMounter() Mounter {
	return &NodeMounter{
		Interface: mount.New(""),
	}
}

func (mounter *NodeMounter) GetDeviceName(mountPath string) (string, int, error) {
	return mount.GetDeviceNameFromMount(mounter, mountPath)
}

// Mount mounts source to target as fstype with given options. 'source' and 'fstype' must
// be an empty string in case it's not required, e.g. for remount, or for auto filesystem
// type, where kernel handles fstype for you. The mount 'options' is a list of options,
// currently come from mount(8), e.g. "ro", "remount", "bind", etc. If no more option is
// required, call Mount with an empty string list or nil.
func (mounter *NodeMounter) Mount(source string, target string, fstype string, options []string) error {
	return mounter.MountSensitive(source, target, fstype, options, nil)
}

// MountSensitive is the same as Mount() but this method allows
// sensitiveOptions to be passed in a separate parameter from the normal
// mount options and ensures the sensitiveOptions are never logged. This
// method should be used by callers that pass sensitive material (like
// passwords) as mount options.
func (mounter *NodeMounter) MountSensitive(source string, target string, fstype string, options []string, sensitiveOptions []string) error {
	// Path to mounter binary if containerized mounter is needed. Otherwise, it is set to empty.
	// All Linux distros are expected to be shipped with a mount utility that a support bind mounts.
	bind, bindOpts, bindRemountOpts, bindRemountOptsSensitive := mount.MakeBindOptsSensitive(options, sensitiveOptions)
	if bind {
		err := mounter.doMount(defaultMountCommand, source, source, target, fstype, bindOpts, bindRemountOptsSensitive, nil)
		if err != nil {
			return err
		}
		return mounter.doMount(defaultMountCommand, source, source, target, fstype, bindRemountOpts, bindRemountOptsSensitive, nil)
	}

	return mounter.doMount(defaultMountCommand, source, source, target, fstype, options, sensitiveOptions, nil)
}

func (mounter *NodeMounter) MountSensitive2(source string, sourceMasked string, target string, fstype string, options []string, sensitiveOptions []string, stdinValues []string) error {
	// Path to mounter binary if containerized mounter is needed. Otherwise, it is set to empty.
	// All Linux distros are expected to be shipped with a mount utility that a support bind mounts.
	bind, bindOpts, bindRemountOpts, bindRemountOptsSensitive := mount.MakeBindOptsSensitive(options, sensitiveOptions)
	if bind {
		err := mounter.doMount(defaultMountCommand, source, sourceMasked, target, fstype, bindOpts, bindRemountOptsSensitive, stdinValues)
		if err != nil {
			return err
		}
		return mounter.doMount(defaultMountCommand, source, sourceMasked, target, fstype, bindRemountOpts, bindRemountOptsSensitive, stdinValues)
	}

	return mounter.doMount(defaultMountCommand, source, sourceMasked, target, fstype, options, sensitiveOptions, stdinValues)
}

func (mounter *NodeMounter) ensureMtab() error {
	// creating /etc/mtab
	pathExists, pathErr := PathExists("/etc/mtab")
	if pathErr != nil {
		klog.Errorf("File /etc/mtab not accessible: %v", pathErr)
		return xerrors.Errorf("file /etc/mtab not accessible: %w", pathErr)
	}

	if !pathExists {
		klog.V(4).Info("Creating /etc/mtab")
		symlinkErr := os.Symlink("/proc/mounts", "/etc/mtab")
		if symlinkErr != nil {
			klog.Errorf("Symlink failed: %v", symlinkErr)
			return xerrors.Errorf("symlink failed: %v", symlinkErr)
		}
	}
	return nil
}

// doMount runs the mount command. mounterPath is the path to mounter binary if containerized mounter is used.
// sensitiveOptions is an extension of options except they will not be logged (because they may contain sensitive material)
func (mounter *NodeMounter) doMount(mountCmd string, source string, sourceMasked string, target string, fstype string, options []string, sensitiveOptions []string, stdinValues []string) error {
	mountArgs, mountArgsLogStr := MakeMountArgsSensitive(source, sourceMasked, target, fstype, options, sensitiveOptions)

	// Ensure /etc/mtab, this is requred by file system client
	// in containerd runtime, /etc/mtab file is missing
	err := mounter.ensureMtab()
	if err != nil {
		return err
	}

	// Logging with sensitive mount options removed.
	klog.V(4).Infof("Mounting cmd (%s) with arguments (%s)", mountCmd, mountArgsLogStr)
	command := exec.Command(mountCmd, mountArgs...)

	if stdinValues != nil {
		stdin, err := command.StdinPipe()
		if err != nil {
			klog.Errorf("Accessing stdin failed: %v\nMounting command: %s\nMounting arguments: %s", err, mountCmd, mountArgsLogStr)
			return xerrors.Errorf("accessing stdin failed, Mounting command '%s', Mounting arguments '%s': %w", mountCmd, mountArgsLogStr, err)
		}

		for _, stdinValue := range stdinValues {
			io.WriteString(stdin, stdinValue)
			io.WriteString(stdin, "\n")
		}
		stdin.Close()
	}

	output, err := command.CombinedOutput()
	if err != nil {
		klog.Errorf("Mount failed: %v\nMounting command: %s\nMounting arguments: %s\nOutput: %s", err, mountCmd, mountArgsLogStr, string(output))
		return xerrors.Errorf("mount failed, Mounting command '%s', Mounting arguments '%s', Output '%s': %w", mountCmd, mountArgsLogStr, string(output), err)
	}
	return err
}

// MakeMountArgsSensitive makes the arguments to the mount(8) command.
// sensitiveOptions is an extension of options except they will not be logged (because they may contain sensitive material)
func MakeMountArgsSensitive(source, sourceMasked, target, fstype string, options []string, sensitiveOptions []string) (mountArgs []string, mountArgsLogStr string) {
	// Build mount command as follows:
	//   mount [-t $fstype] [-o $options] [$source] $target
	mountArgs = []string{}
	mountArgsLogStr = ""
	if len(fstype) > 0 {
		mountArgs = append(mountArgs, "-t", fstype)
		mountArgsLogStr += strings.Join(mountArgs, " ")
	}
	if len(options) > 0 || len(sensitiveOptions) > 0 {
		combinedOptions := []string{}
		combinedOptions = append(combinedOptions, options...)
		combinedOptions = append(combinedOptions, sensitiveOptions...)
		mountArgs = append(mountArgs, "-o", strings.Join(combinedOptions, ","))
		// exclude sensitiveOptions from log string
		mountArgsLogStr += " -o " + sanitizedOptionsForLogging(options, sensitiveOptions)
	}
	if len(source) > 0 {
		mountArgs = append(mountArgs, source)
		mountArgsLogStr += " " + sourceMasked
	}
	mountArgs = append(mountArgs, target)
	mountArgsLogStr += " " + target

	return mountArgs, mountArgsLogStr
}

// sanitizedOptionsForLogging will return a comma separated string containing
// options and sensitiveOptions. Each entry in sensitiveOptions will be
// replaced with the string sensitiveOptionsRemoved
// e.g. o1,o2,<masked>,<masked>
func sanitizedOptionsForLogging(options []string, sensitiveOptions []string) string {
	separator := ""
	if len(options) > 0 && len(sensitiveOptions) > 0 {
		separator = ","
	}

	sensitiveOptionsStart := ""
	sensitiveOptionsEnd := ""
	if len(sensitiveOptions) > 0 {
		sensitiveOptionsStart = strings.Repeat(sensitiveOptionsRemoved+",", len(sensitiveOptions)-1)
		sensitiveOptionsEnd = sensitiveOptionsRemoved
	}

	return strings.Join(options, ",") +
		separator +
		sensitiveOptionsStart +
		sensitiveOptionsEnd
}

// Unmount unmounts the target.
func (mounter *NodeMounter) Unmount(target string) error {
	klog.V(4).Infof("Unmounting %s", target)
	command := exec.Command("umount", target)
	output, err := command.CombinedOutput()
	if err != nil {
		return xerrors.Errorf("umount failed, Unmounting arguments '%s', Output '%s': %w", target, string(output), err)
	}
	return nil
}

// FuseUnmount unmounts the fuse target.
func (mounter *NodeMounter) FuseUnmount(target string, lazy bool) error {
	klog.V(4).Infof("Unmounting %s", target)

	cmdArgs := []string{"-u"}
	if lazy {
		cmdArgs = append(cmdArgs, "-z")
	}

	cmdArgs = append(cmdArgs, target)

	command := exec.Command("fusermount", cmdArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		return xerrors.Errorf("fusermount failed, Unmounting arguments '%s', Output '%s': %w", target, string(output), err)
	}
	return nil
}

// List returns a list of all mounted filesystems.
func (mounter *NodeMounter) List() ([]mount.MountPoint, error) {
	return ListProcMounts(procMountsPath)
}

// ListProcMounts is shared with NsEnterMounter
func ListProcMounts(mountFilePath string) ([]mount.MountPoint, error) {
	content, err := utilio.ConsistentRead(mountFilePath, maxListTries)
	if err != nil {
		return nil, err
	}
	return parseProcMounts(content)
}

func parseProcMounts(content []byte) ([]mount.MountPoint, error) {
	out := []mount.MountPoint{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			// the last split() item is empty string following the last \n
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != expectedNumFieldsPerLine {
			// Do not log line in case it contains sensitive Mount options
			return nil, xerrors.Errorf("wrong number of fields (expected %d, got %d)", expectedNumFieldsPerLine, len(fields))
		}

		mp := mount.MountPoint{
			Device: fields[0],
			Path:   fields[1],
			Type:   fields[2],
			Opts:   strings.Split(fields[3], ","),
		}

		freq, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}
		mp.Freq = freq

		pass, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, err
		}
		mp.Pass = pass

		out = append(out, mp)
	}
	return out, nil
}

// IsLikelyNotMountPoint determines if a directory is not a mountpoint.
// It is fast but not necessarily ALWAYS correct. If the path is in fact
// a bind mount from one part of a mount to another it will not be detected.
// It also can not distinguish between mountpoints and symbolic links.
// mkdir /tmp/a /tmp/b; mount --bind /tmp/a /tmp/b; IsLikelyNotMountPoint("/tmp/b")
// will return true. When in fact /tmp/b is a mount point. If this situation
// is of interest to you, don't use this function...
func (mounter *NodeMounter) IsLikelyNotMountPoint(file string) (bool, error) {
	stat, err := os.Stat(file)
	if err != nil {
		return true, err
	}
	rootStat, err := os.Stat(filepath.Dir(strings.TrimSuffix(file, "/")))
	if err != nil {
		return true, err
	}
	// If the directory has a different device as parent, then it is a mountpoint.
	if stat.Sys().(*syscall.Stat_t).Dev != rootStat.Sys().(*syscall.Stat_t).Dev {
		return false, nil
	}

	return true, nil
}

// GetMountRefs finds all mount references to pathname, returns a
// list of paths. Path could be a mountpoint or a normal
// directory (for bind mount).
func (mounter *NodeMounter) GetMountRefs(pathname string) ([]string, error) {
	pathExists, pathErr := PathExists(pathname)
	if !pathExists {
		return []string{}, nil
	} else if IsCorruptedMount(pathErr) {
		klog.Warningf("GetMountRefs found corrupted mount at %s, treating as unmounted path", pathname)
		return []string{}, nil
	} else if pathErr != nil {
		return nil, xerrors.Errorf("error checking path %s: %w", pathname, pathErr)
	}
	realpath, err := filepath.EvalSymlinks(pathname)
	if err != nil {
		return nil, err
	}
	return SearchMountPoints(realpath, procMountInfoPath)
}

// SearchMountPoints finds all mount references to the source, returns a list of
// mountpoints.
// The source can be a mount point or a normal directory (bind mount). We
// didn't support device because there is no use case by now.
// Some filesystems may share a source name, e.g. tmpfs. And for bind mounting,
// it's possible to mount a non-root path of a filesystem, so we need to use
// root path and major:minor to represent mount source uniquely.
// This implementation is shared between Linux and NsEnterMounter
func SearchMountPoints(hostSource, mountInfoPath string) ([]string, error) {
	mis, err := ParseMountInfo(mountInfoPath)
	if err != nil {
		return nil, err
	}

	mountID := 0
	rootPath := ""
	major := -1
	minor := -1

	// Finding the underlying root path and major:minor if possible.
	// We need search in backward order because it's possible for later mounts
	// to overlap earlier mounts.
	for i := len(mis) - 1; i >= 0; i-- {
		if hostSource == mis[i].MountPoint || mount.PathWithinBase(hostSource, mis[i].MountPoint) {
			// If it's a mount point or path under a mount point.
			mountID = mis[i].ID
			rootPath = filepath.Join(mis[i].Root, strings.TrimPrefix(hostSource, mis[i].MountPoint))
			major = mis[i].Major
			minor = mis[i].Minor
			break
		}
	}

	if rootPath == "" || major == -1 || minor == -1 {
		return nil, xerrors.Errorf("failed to get root path and major-minor for %s", hostSource)
	}

	var refs []string
	for i := range mis {
		if mis[i].ID == mountID {
			// Ignore mount entry for mount source itself.
			continue
		}
		if mis[i].Root == rootPath && mis[i].Major == major && mis[i].Minor == minor {
			refs = append(refs, mis[i].MountPoint)
		}
	}

	return refs, nil
}

// PathExists returns true if the specified path exists.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else if IsCorruptedMount(err) {
		return true, err
	}
	return false, err
}

// MakeDir creates a dir for the path
func MakeDir(path string) error {
	err := os.MkdirAll(path, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func hasValueInArray(arr []string, val string) bool {
	for _, arrVal := range arr {
		if arrVal == val {
			return true
		}
	}
	return false
}

// GetMountOptions returns mount options from VolumeCapability_MountVolume
func GetMountOptions(volumes *csi.VolumeCapability_MountVolume) []string {
	mountOptions := []string{}
	if volumes != nil {
		for _, f := range volumes.MountFlags {
			if !hasValueInArray(mountOptions, f) {
				mountOptions = append(mountOptions, f)
			}
		}
	}
	return mountOptions
}
