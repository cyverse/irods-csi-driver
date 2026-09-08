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
	config := irodsclient_connection.IRODSConnectionConfig{
		ConnectTimeout:  60 * time.Second,
		ApplicationName: applicationName,
	}
	irodsConn, err := irodsclient_connection.NewIRODSConnection(account, &config)
	if err != nil {
		klog.V(5).Infof("Failed to create an iRODS connection - %v", conn.ToIRODSAccount().GetRedacted())
		return err
	}

	err = irodsConn.Connect()
	if err != nil {
		klog.V(5).Infof("Failed to connect to iRODS - %v", conn.ToIRODSAccount().GetRedacted())
		return err
	}

	irodsConn.Disconnect()
	return nil
}
