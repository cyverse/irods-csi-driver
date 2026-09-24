package irods

import (
	"time"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsclient_connection "github.com/cyverse/go-irodsclient/irods/connection"
	irodsclient_types "github.com/cyverse/go-irodsclient/irods/types"
	irodsfs_pool_client "github.com/cyverse/irodsfs-pool/client"
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

// TestConnection tests logging in to the configured iRODS service. When a pool
// endpoint is configured, it tests the same pool-service login path used by
// the mount instead of connecting to iRODS directly.
func TestConnection(conn *IRODSFSConnectionInfo) error {
	account := GetIRODSAccount(conn)
	if conn.PoolEndpoint != "" {
		poolClient := irodsfs_pool_client.NewPoolServiceClient(conn.PoolEndpoint, 60*time.Second, false, "", nil)
		if err := poolClient.Connect(); err != nil {
			klog.V(5).Infof("Failed to connect to iRODS pool service %q", conn.PoolEndpoint)
			return err
		}
		defer poolClient.Disconnect()

		session, err := poolClient.NewSession(account, applicationName, "connection test")
		if err != nil {
			klog.V(5).Infof("Failed to log in through iRODS pool service %q - %v", conn.PoolEndpoint, account.GetRedacted())
			return err
		}
		if err := session.Release(); err != nil {
			klog.V(5).Infof("Failed to release iRODS pool service session %q - %v", conn.PoolEndpoint, account.GetRedacted())
			return err
		}

		return nil
	}

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
