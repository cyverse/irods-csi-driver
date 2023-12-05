package irods

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsclient_types "github.com/cyverse/go-irodsclient/irods/types"
	irodsfs_common_inode "github.com/cyverse/irodsfs-common/inode"
	irodsfs_common_irods "github.com/cyverse/irodsfs-common/irods"
	irodsfs_common_vpath "github.com/cyverse/irodsfs-common/vpath"
	"github.com/pkg/xattr"
	"golang.org/x/xerrors"
	"k8s.io/klog"
)

const (
	overlayFSOpaqueXAttr string = "trusted.overlay.opaque"
)

// OverlayFSSyncher is a struct for OverlayFSSyncher
type OverlayFSSyncher struct {
	irodsConnectionInfo *IRODSFSConnectionInfo
	irodsFsClient       *irodsclient_fs.FileSystem
	irodsFsVPathManager *irodsfs_common_vpath.VPathManager
	upperLayerPath      string
}

// NewOverlayFSSyncher creates a new OverlayFSSyncher
func NewOverlayFSSyncher(irodsConnInfo *IRODSFSConnectionInfo, upper string) (*OverlayFSSyncher, error) {
	irodsAccount, err := GetIRODSAccount(irodsConnInfo)
	if err != nil {
		return nil, xerrors.Errorf("failed to get irods account: %w", err)
	}

	fsConfig := GetIRODSFilesystemConfig()

	fsClient, err := irodsfs_common_irods.NewIRODSFSClientDirect(irodsAccount, fsConfig)
	if err != nil {
		return nil, xerrors.Errorf("failed to create fs client: %w", err)
	}

	fsClientDirect := fsClient.(*irodsfs_common_irods.IRODSFSClientDirect)

	inodeManager := irodsfs_common_inode.NewInodeManager()

	vpathManager, err := irodsfs_common_vpath.NewVPathManager(fsClient, inodeManager, irodsConnInfo.PathMappings)
	if err != nil {
		return nil, xerrors.Errorf("failed to create Virtual Path Manager: %w", err)
	}

	absUpper, err := filepath.Abs(upper)
	if err != nil {
		return nil, xerrors.Errorf("failed to get abs upper path for %q: %w", upper, err)
	}

	return &OverlayFSSyncher{
		irodsConnectionInfo: irodsConnInfo,
		irodsFsClient:       fsClientDirect.GetFSClient(),
		irodsFsVPathManager: vpathManager,
		upperLayerPath:      absUpper,
	}, nil
}

func (syncher *OverlayFSSyncher) Release() {
	if syncher.irodsFsClient != nil {
		syncher.irodsFsClient.Release()
		syncher.irodsFsClient = nil
	}
}

// GetUpperLayerPath returns upper layer path
func (syncher *OverlayFSSyncher) GetUpperLayerPath() string {
	return syncher.upperLayerPath
}

// Sync syncs upper layer data to lower layer
func (syncher *OverlayFSSyncher) Sync() error {
	klog.V(5).Infof("sync'ing path %q", syncher.upperLayerPath)

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return xerrors.Errorf("failed to walk %q: %w", path, err)
		}

		if d.IsDir() {
			if path == syncher.upperLayerPath {
				// skip root
				return nil
			}

			syncErr := syncher.syncDir(path)
			if syncErr != nil {
				klog.V(3).Infof("failed to sync dir %q, %s", path, syncErr)
				return nil
			}
		} else {
			// file
			if d.Type()&os.ModeCharDevice != 0 {
				syncErr := syncher.syncWhiteout(path)
				if syncErr != nil {
					klog.V(3).Infof("failed to sync whiteout %q, %s", path, syncErr)
					return nil
				}
			} else {
				syncErr := syncher.syncFile(path)
				if syncErr != nil {
					klog.V(3).Infof("failed to sync file %q, %s", path, syncErr)
					return nil
				}
			}
		}
		return nil
	}

	err := filepath.WalkDir(syncher.upperLayerPath, walkFunc)
	if err != nil {
		return xerrors.Errorf("failed to walk dir %q: %w", syncher.upperLayerPath, err)
	}

	return nil
}

func (syncher *OverlayFSSyncher) getIRODSPath(localPath string) (string, error) {
	relpath, err := filepath.Rel(syncher.upperLayerPath, localPath)
	if err != nil {
		return "", xerrors.Errorf("failed to get relative path from %q to %q", syncher.upperLayerPath, localPath)
	}

	vpath := path.Join("/", relpath)

	entry := syncher.irodsFsVPathManager.GetClosestEntry(vpath)
	if entry == nil {
		return "", xerrors.Errorf("failed to find closest vpath entry for path %q", vpath)
	}

	irodsPath, err := entry.GetIRODSPath(vpath)
	if err != nil {
		return "", xerrors.Errorf("failed to get iRODS path for path %q: %w", vpath, err)
	}

	return irodsPath, nil
}

func (syncher *OverlayFSSyncher) syncWhiteout(path string) error {
	klog.V(5).Infof("processing whiteout file %q", path)

	irodsPath, err := syncher.getIRODSPath(path)
	if err != nil {
		return err
	}

	entry, err := syncher.irodsFsClient.Stat(irodsPath)
	if err != nil {
		if irodsclient_types.IsFileNotFoundError(err) {
			// not exist
			klog.V(5).Infof("file or dir %q not exist", irodsPath)
			// suppress warning
			return nil
		}

		return xerrors.Errorf("failed to stat %q: %w", irodsPath, err)
	}

	klog.V(5).Infof("deleting file or dir %q", irodsPath)

	// remove
	if entry.IsDir() {
		err = syncher.irodsFsClient.RemoveDir(irodsPath, true, true)
		if err != nil {
			return xerrors.Errorf("failed to remove dir %q: %w", irodsPath, err)
		}
	} else {
		err = syncher.irodsFsClient.RemoveFile(irodsPath, true)
		if err != nil {
			return xerrors.Errorf("failed to remove file %q: %w", irodsPath, err)
		}
	}

	return nil
}

func (syncher *OverlayFSSyncher) syncFile(path string) error {
	klog.V(5).Infof("processing new or updated file %q", path)

	irodsPath, err := syncher.getIRODSPath(path)
	if err != nil {
		return err
	}

	entry, err := syncher.irodsFsClient.Stat(irodsPath)
	if err != nil {
		if !irodsclient_types.IsFileNotFoundError(err) {
			return xerrors.Errorf("failed to stat %q: %w", irodsPath, err)
		}
	} else {
		// exist
		// if it is a dir, remove first
		// if it is a file, overwrite
		if entry.IsDir() {
			klog.V(5).Infof("deleting dir %q", irodsPath)

			err = syncher.irodsFsClient.RemoveDir(irodsPath, true, true)
			if err != nil {
				return xerrors.Errorf("failed to remove dir %q: %w", irodsPath, err)
			}
		}
	}

	klog.V(5).Infof("copying file %q", irodsPath)

	// upload the file
	err = syncher.irodsFsClient.UploadFileParallel(path, irodsPath, "", 0, false, nil)
	if err != nil {
		return xerrors.Errorf("failed to upload file %q: %w", irodsPath, err)
	}

	return nil
}

func (syncher *OverlayFSSyncher) syncDir(path string) error {
	klog.V(5).Infof("processing dir %q", path)

	irodsPath, err := syncher.getIRODSPath(path)
	if err != nil {
		return err
	}

	opaqueDir := false
	xattrVal, err := xattr.Get(path, overlayFSOpaqueXAttr)
	if err == nil {
		xattrValStr := string(xattrVal)
		klog.V(5).Infof("xattr for path %q: %q = %q", path, overlayFSOpaqueXAttr, xattrValStr)

		if strings.ToLower(xattrValStr) == "y" {
			opaqueDir = true
		}
	}

	entry, err := syncher.irodsFsClient.Stat(irodsPath)
	if err != nil {
		if irodsclient_types.IsFileNotFoundError(err) {
			// not exist
			klog.V(5).Infof("making dir %q", irodsPath)

			err = syncher.irodsFsClient.MakeDir(irodsPath, true)
			if err != nil {
				return xerrors.Errorf("failed to make dir %q: %w", irodsPath, err)
			}

			return nil
		}

		return xerrors.Errorf("failed to stat %q: %w", path, err)
	}

	// exist
	// if it is a file, remove first
	// if it is a dir, merge or remove
	if !entry.IsDir() {
		// file
		klog.V(5).Infof("deleting file %q", irodsPath)

		err = syncher.irodsFsClient.RemoveFile(irodsPath, true)
		if err != nil {
			return xerrors.Errorf("failed to remove file %q: %w", irodsPath, err)
		}

		klog.V(5).Infof("making dir %q", irodsPath)

		err = syncher.irodsFsClient.MakeDir(irodsPath, true)
		if err != nil {
			return xerrors.Errorf("failed to make dir %q: %w", irodsPath, err)
		}

		return nil
	}

	// merge or remove
	if opaqueDir {
		// remove
		klog.V(5).Infof("emptying dir %q", irodsPath)

		err = syncher.clearDirEntries(irodsPath)
		if err != nil {
			return xerrors.Errorf("failed to clear %q: %w", irodsPath, err)
		}
	} else {
		// merge
		klog.V(5).Infof("merging dir %q", irodsPath)
	}

	return nil
}

func (syncher *OverlayFSSyncher) clearDirEntries(path string) error {
	entries, err := syncher.irodsFsClient.List(path)
	if err != nil {
		return xerrors.Errorf("failed to read dir %q: %w", path, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			err = syncher.irodsFsClient.RemoveDir(entry.Path, true, true)
			if err != nil {
				return xerrors.Errorf("failed to remove dir %q: %w", entry.Path, err)
			}
		} else {
			err = syncher.irodsFsClient.RemoveFile(entry.Path, true)
			if err != nil {
				return xerrors.Errorf("failed to remove %q: %w", entry.Path, err)
			}
		}
	}

	return nil
}
