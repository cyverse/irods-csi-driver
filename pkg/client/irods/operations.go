package irods

import (
	"time"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsclient_connection "github.com/cyverse/go-irodsclient/irods/connection"
	irodsclient_types "github.com/cyverse/go-irodsclient/irods/types"
	"k8s.io/klog"
)

const (
	applicationName string = "irods-csi-driver"
)

// GetIRODSAccount creates a new account
func GetIRODSAccount(conn *IRODSFSConnectionInfo) *irodsclient_types.IRODSAccount {
	return conn.ToIRODSAccount()
}

// GetIRODSFilesystemConfig creates a new filesystem config
func GetIRODSFilesystemConfig() *irodsclient_fs.FileSystemConfig {
	return irodsclient_fs.NewFileSystemConfig(applicationName)
}

// GetIRODSFilesystem creates a new filesystem
func GetIRODSFilesystem(conn *IRODSFSConnectionInfo) (*irodsclient_fs.FileSystem, error) {
	account := GetIRODSAccount(conn)
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
	account := GetIRODSAccount(conn)

	// test connect
	irodsConn := irodsclient_connection.NewIRODSConnection(account, time.Second*60, applicationName)
	err := irodsConn.Connect()
	if err != nil {
		klog.V(5).Infof("Failed to connect to iRODS - %v", conn.ToIRODSAccount().GetRedacted())
		return err
	}

	irodsConn.Disconnect()
	return nil
}
