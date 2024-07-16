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
	authScheme := irodsclient_types.GetAuthScheme(conn.AuthScheme)
	if authScheme == irodsclient_types.AuthSchemeUnknown {
		authScheme = irodsclient_types.AuthSchemeNative
	}

	csNegotiation, err := irodsclient_types.GetCSNegotiationRequire(conn.CSNegotiationPolicy)
	if err != nil {
		return nil, err
	}

	account, err := irodsclient_types.CreateIRODSProxyAccount(conn.Hostname, conn.Port,
		conn.ClientUser, conn.Zone, conn.User, conn.Zone,
		authScheme, conn.Password, conn.Resource)
	if err != nil {
		return nil, err
	}

	// optional for ssl,
	sslConfig, err := irodsclient_types.CreateIRODSSSLConfig(conn.CACertificateFile, conn.CACertificatePath, conn.EncryptionKeySize,
		conn.EncryptionAlgorithm, conn.SaltSize, conn.HashRounds)
	if err != nil {
		return nil, err
	}

	if authScheme == irodsclient_types.AuthSchemePAM {
		account.SetSSLConfiguration(sslConfig)
		account.SetCSNegotiation(true, irodsclient_types.CSNegotiationRequireSSL)
	} else if conn.ClientServerNegotiation {
		account.SetSSLConfiguration(sslConfig)
		account.SetCSNegotiation(conn.ClientServerNegotiation, csNegotiation)
	}

	return account, nil
}

// GetIRODSFilesystemConfig creates a new filesystem config
func GetIRODSFilesystemConfig() *irodsclient_fs.FileSystemConfig {
	return irodsclient_fs.NewFileSystemConfigWithDefault(applicationName)
}

// GetIRODSFilesystem creates a new filesystem
func GetIRODSFilesystem(conn *IRODSFSConnectionInfo) (*irodsclient_fs.FileSystem, error) {
	account, err := GetIRODSAccount(conn)
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
	account, err := GetIRODSAccount(conn)
	if err != nil {
		return err
	}

	// test connect
	irodsConn := irodsclient_connection.NewIRODSConnection(account, 5*time.Minute, applicationName)
	err = irodsConn.Connect()
	if err != nil {
		return err
	}

	irodsConn.Disconnect()
	return nil
}
