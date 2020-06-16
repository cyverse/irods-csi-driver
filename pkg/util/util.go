package util

import (
    "fmt"
    "net/url"
    "os"
    "path"
    "path/filepath"
    "strings"

	"github.com/pkg/errors"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
	"k8s.io/klog"
	"k8s.io/utils/mount"

)

const (
    fuseType = "fuse"
    nfsType = "nfs"
    webdavType = "webdav"
)

// hold the parameters list which can be configured
type Config struct {
	DriverType string // driver type [fuse|nfs|webdav]
	Endpoint string // CSI endpoint

	Version bool // irods csi version
}

// validate the driver type
func ValidateDriverType(driverType string) error {
	if driverType == "" {
		return errors.New("driver type is empty")
	}

    switch(driverType) {
    case fuseType:
        fallthrough
    case nfsType:
        fallthrough
    case webdavType:
        return nil

    default:
        return fmt.Errorf("unknown driver type - %v", driverType)
    }
}

// parse endpoint (TCP or UNIX)
func ParseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("could not parse endpoint: %v", err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
	case "unix":
		addr = path.Join("/", addr)
        err := os.Remove(addr)
		if err != nil && !os.IsNotExist(err) {
			return "", "", fmt.Errorf("could not remove unix domain socket %q: %v", addr, err)
		}
	default:
		return "", "", fmt.Errorf("unsupported protocol: %s", scheme)
	}

	return scheme, addr, nil
}



/////////////// UNUSED YET




// CreateMountPoint creates the directory with given path
func CreateMountPoint(mountPath string) error {
	return os.MkdirAll(mountPath, 0750)
}

// CheckDirExists checks directory  exists or not
func CheckDirExists(p string) bool {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return false
	}
	return true
}

// IsMountPoint checks if the given path is mountpoint or not
func IsMountPoint(p string) (bool, error) {
	dummyMount := mount.New("")
	notMnt, err := dummyMount.IsLikelyNotMountPoint(p)
	if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}

	return !notMnt, nil
}

// Mount mounts the source to target path
func Mount(source, target, fstype string, options []string) error {
	dummyMount := mount.New("")
	return dummyMount.Mount(source, target, fstype, options)
}
