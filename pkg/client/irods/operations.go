package irods

import (
	"time"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsclient_connection "github.com/cyverse/go-irodsclient/irods/connection"
	irodsclient_types "github.com/cyverse/go-irodsclient/irods/types"
)

const (
	applicationName string = "irods-csi-driver"
)

// GetIRODSAccount creates a new account
func GetIRODSAccount(conn *IRODSFSConnectionInfo) (*irodsclient_types.IRODSAccount, error) {
	return irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, irodsclient_types.AuthSchemeNative, conn.Password, conn.Resource)
}

// GetIRODSFilesystemConfig creates a new filesystem config
func GetIRODSFilesystemConfig() *irodsclient_fs.FileSystemConfig {
	return irodsclient_fs.NewFileSystemConfigWithDefault(applicationName)
}

// GetIRODSFilesystem creates a new filesystem
func GetIRODSFilesystem(conn *IRODSFSConnectionInfo) (*irodsclient_fs.FileSystem, error) {
	account, err := irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, irodsclient_types.AuthSchemeNative, conn.Password, conn.Resource)
	if err != nil {
		return nil, err
	}

	return irodsclient_fs.NewFileSystemWithDefault(account, applicationName)
}

// Mkdir creates a new directory
func Mkdir(conn *IRODSFSConnectionInfo, path string) error {
	filesystem, err := GetIRODSFilesystem(conn)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return filesystem.MakeDir(path, true)
}

// Rmdir deletes a directory
func Rmdir(conn *IRODSFSConnectionInfo, path string) error {
	filesystem, err := GetIRODSFilesystem(conn)
	if err != nil {
		return err
	}

	defer filesystem.Release()

	return filesystem.RemoveDir(path, true, true)
}

// TestConnection just test connection creation
func TestConnection(conn *IRODSFSConnectionInfo) error {
	account, err := irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port, conn.ClientUser, conn.Zone, conn.User, conn.Zone, irodsclient_types.AuthSchemeNative, conn.Password, conn.Resource)
	if err != nil {
		return err
	}

	irodsConn := irodsclient_connection.NewIRODSConnection(account, 5*time.Minute, applicationName)
	err = irodsConn.Connect()
	if err != nil {
		return err
	}

	irodsConn.Disconnect()
	return nil
}
