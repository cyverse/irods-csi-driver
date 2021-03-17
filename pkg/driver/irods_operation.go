package driver

import (
	"time"

	"github.com/cyverse/go-irodsclient/fs"
	"github.com/cyverse/go-irodsclient/irods/types"
)

const (
	applicationName string = "irods-csi-driver"
)

// IRODSMkdir creates a new directory
func IRODSMkdir(conn *IRODSConnectionInfo, path string) error {
	account, err := types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, types.AuthSchemeNative, conn.Password)
	if err != nil {
		return err
	}

	filesystem, err := fs.NewFileSystemWithDefault(account, applicationName)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return filesystem.MakeDir(path, true)
}

// IRODSRmdir deletes a directory
func IRODSRmdir(conn *IRODSConnectionInfo, path string) error {
	account, err := types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, types.AuthSchemeNative, conn.Password)
	if err != nil {
		return err
	}

	filesystem, err := fs.NewFileSystemWithDefault(account, applicationName)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return filesystem.RemoveDir(path, true, true)
}

// IRODSTestConnection just test connection creation
func IRODSTestConnection(conn *IRODSConnectionInfo) error {
	account, err := types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, types.AuthSchemeNative, conn.Password)
	if err != nil {
		return err
	}

	oneMin := 1 * time.Minute
	fsConfig := fs.NewFileSystemConfig(applicationName, oneMin, oneMin, 1, oneMin, oneMin)
	filesystem, err := fs.NewFileSystem(account, fsConfig)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return nil
}
