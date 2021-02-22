package driver

import (
	"fmt"
	"strings"

	"github.com/cyverse/go-irodsclient/fs"
	"github.com/cyverse/go-irodsclient/irods/types"
)

const (
	applicationName string = "irods-csi-driver"
)

func makeIRODSZonePath(zone string, path string) string {
	// argument path may not start with zone
	inputPath := path
	zonePath := fmt.Sprintf("/%s/", zone)

	if !strings.HasPrefix(path, zonePath) {
		if strings.HasPrefix(path, "/") {
			inputPath = fmt.Sprintf("/%s%s", zone, path)
		} else {
			inputPath = fmt.Sprintf("/%s/%s", zone, path)
		}
	}
	return inputPath
}

// IRODSMkdir creates a new directory
func IRODSMkdir(conn *IRODSConnection, path string) error {
	inputPath := makeIRODSZonePath(conn.Zone, path)

	account, err := types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, types.AuthSchemeNative, conn.Password)
	if err != nil {
		return err
	}

	filesystem := fs.NewFileSystemWithDefault(account, applicationName)
	defer filesystem.Release()

	return filesystem.MakeDir(inputPath, true)
}

// IRODSRmdir deletes a directory
func IRODSRmdir(conn *IRODSConnection, path string) error {
	inputPath := makeIRODSZonePath(conn.Zone, path)

	account, err := types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, types.AuthSchemeNative, conn.Password)
	if err != nil {
		return err
	}

	filesystem := fs.NewFileSystemWithDefault(account, applicationName)
	defer filesystem.Release()

	return filesystem.RemoveDir(inputPath, true, true)
}
