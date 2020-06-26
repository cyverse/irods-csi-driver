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
	"os"

	"k8s.io/utils/mount"
)

type Mounter interface {
	mount.Interface
	MakeDir(pathname string) error
    GetDeviceName(mountPath string) (string, int, error)
}

type NodeMounter struct {
	mount.Interface
}

func newNodeMounter() Mounter {
	return &NodeMounter{
		Interface: mount.New(""),
	}
}

func (mounter *NodeMounter) MakeDir(pathname string) error {
	err := os.MkdirAll(pathname, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
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
		err := mounter.doMount("", defaultMountCommand, source, target, fstype, bindOpts, bindRemountOptsSensitive)
		if err != nil {
			return err
		}
		return mounter.doMount("", defaultMountCommand, source, target, fstype, bindRemountOpts, bindRemountOptsSensitive)
	}

	return mounter.doMount(mounter.mounterPath, defaultMountCommand, source, target, fstype, options, sensitiveOptions)
}
