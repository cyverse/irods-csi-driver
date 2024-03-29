/*
Following functions or objects are from the code under APL2 License.
- IsCorruptedMount
- MountInfo
- ParseMountInfo
Original code:
- https://github.com/kubernetes/utils/blob/master/mount/mount_helper_unix.go


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
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/xerrors"
	utilio "k8s.io/utils/io"
)

const (
	// At least number of fields per line in /proc/<pid>/mountinfo.
	expectedAtLeastNumFieldsPerMountInfo = 10
	// How many times to retry for a consistent read of /proc/mounts.
	maxListTries = 3
)

// IsCorruptedMount return true if err is about corrupted mount point
func IsCorruptedMount(err error) bool {
	if err == nil {
		return false
	}
	var underlyingError error
	switch pe := err.(type) {
	case nil:
		return false
	case *os.PathError:
		underlyingError = pe.Err
	case *os.LinkError:
		underlyingError = pe.Err
	case *os.SyscallError:
		underlyingError = pe.Err
	}

	return underlyingError == syscall.ENOTCONN || underlyingError == syscall.ESTALE || underlyingError == syscall.EIO || underlyingError == syscall.EACCES
}

// MountInfo represents a single line in /proc/<pid>/mountinfo.
type MountInfo struct { // nolint: golint
	// Unique ID for the mount (maybe reused after umount).
	ID int
	// The ID of the parent mount (or of self for the root of this mount namespace's mount tree).
	ParentID int
	// Major indicates one half of the device ID which identifies the device class
	// (parsed from `st_dev` for files on this filesystem).
	Major int
	// Minor indicates one half of the device ID which identifies a specific
	// instance of device (parsed from `st_dev` for files on this filesystem).
	Minor int
	// The pathname of the directory in the filesystem which forms the root of this mount.
	Root string
	// Mount source, filesystem-specific information. e.g. device, tmpfs name.
	Source string
	// Mount point, the pathname of the mount point.
	MountPoint string
	// Optional fieds, zero or more fields of the form "tag[:value]".
	OptionalFields []string
	// The filesystem type in the form "type[.subtype]".
	FsType string
	// Per-mount options.
	MountOptions []string
	// Per-superblock options.
	SuperOptions []string
}

// ParseMountInfo parses /proc/xxx/mountinfo.
func ParseMountInfo(filename string) ([]MountInfo, error) {
	content, err := utilio.ConsistentRead(filename, maxListTries)
	if err != nil {
		return []MountInfo{}, err
	}
	contentStr := string(content)
	infos := []MountInfo{}

	for _, line := range strings.Split(contentStr, "\n") {
		if line == "" {
			// the last split() item is empty string following the last \n
			continue
		}
		// See `man proc` for authoritative description of format of the file.
		fields := strings.Fields(line)
		if len(fields) < expectedAtLeastNumFieldsPerMountInfo {
			return nil, xerrors.Errorf("wrong number of fields in (expected at least %d, got %d), line %q", expectedAtLeastNumFieldsPerMountInfo, len(fields), line)
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, xerrors.Errorf("failed to atoi %q: %w", fields[0], err)
		}
		parentID, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, xerrors.Errorf("failed to atoi %q: %w", fields[1], err)
		}
		mm := strings.Split(fields[2], ":")
		if len(mm) != 2 {
			return nil, xerrors.Errorf("failed to parse %q, unexpected minor-major pair %v", fields[2], mm)
		}
		major, err := strconv.Atoi(mm[0])
		if err != nil {
			return nil, xerrors.Errorf("failed to parse %q, unalbe to parse major device id: %w", mm[0], err)
		}
		minor, err := strconv.Atoi(mm[1])
		if err != nil {
			return nil, xerrors.Errorf("failed to parse %q, unable to parse minor device id: %w", mm[1], err)
		}

		info := MountInfo{
			ID:           id,
			ParentID:     parentID,
			Major:        major,
			Minor:        minor,
			Root:         fields[3],
			MountPoint:   fields[4],
			MountOptions: strings.Split(fields[5], ","),
		}
		// All fields until "-" are "optional fields".
		i := 6
		for ; i < len(fields) && fields[i] != "-"; i++ {
			info.OptionalFields = append(info.OptionalFields, fields[i])
		}
		// Parse the rest 3 fields.
		i++
		if len(fields)-i < 3 {
			return nil, xerrors.Errorf("expect 3 fields in %q, got %d", line, len(fields)-i)
		}
		info.FsType = fields[i]
		info.Source = fields[i+1]
		info.SuperOptions = strings.Split(fields[i+2], ",")
		infos = append(infos, info)
	}
	return infos, nil
}
