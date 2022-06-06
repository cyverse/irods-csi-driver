package driver

import (
	"time"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsclient_types "github.com/cyverse/go-irodsclient/irods/types"
)

const (
	applicationName string = "irods-csi-driver"
)

// IRODSMkdir creates a new directory
func IRODSMkdir(conn *IRODSConnectionInfo, path string) error {
	account, err := irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, irodsclient_types.AuthSchemeNative, conn.Password, conn.Resource)
	if err != nil {
		return err
	}

	filesystem, err := irodsclient_fs.NewFileSystemWithDefault(account, applicationName)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return filesystem.MakeDir(path, true)
}

// IRODSRmdir deletes a directory
func IRODSRmdir(conn *IRODSConnectionInfo, path string) error {
	account, err := irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, irodsclient_types.AuthSchemeNative, conn.Password, conn.Resource)
	if err != nil {
		return err
	}

	filesystem, err := irodsclient_fs.NewFileSystemWithDefault(account, applicationName)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return filesystem.RemoveDir(path, true, true)
}

// IRODSTestConnection just test connection creation
func IRODSTestConnection(conn *IRODSConnectionInfo) error {
	account, err := irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, irodsclient_types.AuthSchemeNative, conn.Password, conn.Resource)
	if err != nil {
		return err
	}

	oneMin := 1 * time.Minute
	oneHour := 1 * time.Hour
	fsConfig := irodsclient_fs.NewFileSystemConfig(applicationName, oneHour, oneMin, oneMin, 1, oneMin, oneMin, nil, true, false)
	filesystem, err := irodsclient_fs.NewFileSystem(account, fsConfig)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return nil
}
