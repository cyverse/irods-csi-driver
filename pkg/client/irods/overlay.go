package irods

import (
	"fmt"
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
	syncStatusFileSuffix string = ".csi.overlay.sync.log"
)

// OverlayFSSyncher is a struct for OverlayFSSyncher
type OverlayFSSyncher struct {
	volumeID            string
	irodsConnectionInfo *IRODSFSConnectionInfo
	irodsFsClient       *irodsclient_fs.FileSystem
	irodsFsVPathManager *irodsfs_common_vpath.VPathManager
	parallelJobManager  *ParallelJobManager
	upperLayerPath      string
}

// NewOverlayFSSyncher creates a new OverlayFSSyncher
func NewOverlayFSSyncher(volumeID string, irodsConnInfo *IRODSFSConnectionInfo, upper string) (*OverlayFSSyncher, error) {
	irodsAccount := GetIRODSAccount(irodsConnInfo)
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

	parallelJobManager := NewParallelJobManager(4)

	absUpper, err := filepath.Abs(upper)
	if err != nil {
		return nil, xerrors.Errorf("failed to get abs upper path for %q: %w", upper, err)
	}

	return &OverlayFSSyncher{
		volumeID:            volumeID,
		irodsConnectionInfo: irodsConnInfo,
		irodsFsClient:       fsClientDirect.GetFSClient(),
		irodsFsVPathManager: vpathManager,
		parallelJobManager:  parallelJobManager,
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

func (syncher *OverlayFSSyncher) getStatusFilePath() string {
	return fmt.Sprintf("/%s/home/%s/.%s%s", syncher.irodsConnectionInfo.ClientZoneName, syncher.irodsConnectionInfo.ClientUsername, syncher.volumeID, syncStatusFileSuffix)
}

// Sync syncs upper layer data to lower layer
func (syncher *OverlayFSSyncher) Sync() error {
	klog.V(5).Infof("sync'ing path %q for volume %q", syncher.upperLayerPath, syncher.volumeID)

	entries, err := os.ReadDir(syncher.upperLayerPath)
	if err != nil {
		return xerrors.Errorf("failed to read dir entires for upperdir %q, volume %q: %w", syncher.upperLayerPath, syncher.volumeID, err)
	}

	if len(entries) == 0 {
		klog.V(5).Infof("stop sync'ing path %q for volume %q, no updates to sync", syncher.upperLayerPath, syncher.volumeID)
		return nil
	}

	syncher.parallelJobManager.Start()

	statusFile := syncher.getStatusFilePath()

	// create status file
	statusFileHandle, err := syncher.irodsFsClient.OpenFile(statusFile, "", "w+") // write + truncate
	if err != nil {
		return xerrors.Errorf("failed to create sync status file %q: %w", statusFile, err)
	}
	defer statusFileHandle.Close()

	currentDirPath := ""

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return xerrors.Errorf("failed to walk %q for volume %q: %w", path, syncher.volumeID, err)
		}

		parentDirPath := filepath.Dir(path)
		if currentDirPath != parentDirPath {
			// new dir
			// wait until other jobs are done
			taskName := fmt.Sprintf("barrier - %q for volume %q", parentDirPath, syncher.volumeID)
			scheduleErr := syncher.parallelJobManager.ScheduleBarrier(taskName)
			if scheduleErr != nil {
				klog.Errorf("failed to schedule barrier task for %q, volume %q, %s", path, syncher.volumeID, scheduleErr)
				return nil
			}

			currentDirPath = parentDirPath
		}

		if d.IsDir() {
			if path == syncher.upperLayerPath {
				// skip root
				return nil
			}

			dirSyncTask := func(job *ParallelJob) error {
				syncErr := syncher.syncDir(path, statusFileHandle)
				if syncErr != nil {
					klog.Errorf("failed to sync dir %q, volume %q, %s", path, syncher.volumeID, syncErr)
					return nil
				}
				return nil
			}

			taskName := fmt.Sprintf("sync dir - %q for volume %q", path, syncher.volumeID)
			scheduleErr := syncher.parallelJobManager.Schedule(taskName, dirSyncTask, 1)
			if scheduleErr != nil {
				klog.Errorf("failed to schedule dir sync task for %q, volume %q, %s", path, syncher.volumeID, scheduleErr)
				return nil
			}
		} else {
			// file
			if syncher.isIgnoredFile(path) {
				return nil
			}

			if d.Type()&os.ModeCharDevice != 0 {
				whiteoutSyncTask := func(job *ParallelJob) error {
					syncErr := syncher.syncWhiteout(path, statusFileHandle)
					if syncErr != nil {
						klog.Errorf("failed to sync whiteout %q, volume %q, %s", path, syncher.volumeID, syncErr)
						return nil
					}
					return nil
				}

				taskName := fmt.Sprintf("sync whiteout - %q for volume %q, ", path, syncher.volumeID)
				scheduleErr := syncher.parallelJobManager.Schedule(taskName, whiteoutSyncTask, 1)
				if scheduleErr != nil {
					klog.Errorf("failed to schedule whiteout sync task for %q, volume %q, %s", path, syncher.volumeID, scheduleErr)
					return nil
				}
			} else {
				fileSyncTask := func(job *ParallelJob) error {
					syncErr := syncher.syncFile(path, statusFileHandle)
					if syncErr != nil {
						klog.Errorf("failed to sync file %q, volume %q, %s", path, syncher.volumeID, syncErr)
						return nil
					}
					return nil
				}

				taskName := fmt.Sprintf("sync file - %q for volume %q", path, syncher.volumeID)
				scheduleErr := syncher.parallelJobManager.Schedule(taskName, fileSyncTask, 1)
				if scheduleErr != nil {
					klog.Errorf("failed to schedule file sync task for %q, volume %q, %s", path, syncher.volumeID, scheduleErr)
					return nil
				}
			}
		}
		return nil
	}

	err = filepath.WalkDir(syncher.upperLayerPath, walkFunc)
	if err != nil {
		return xerrors.Errorf("failed to walk dir %q for volume %q: %w", syncher.upperLayerPath, syncher.volumeID, err)
	}

	syncher.parallelJobManager.DoneScheduling()
	err = syncher.parallelJobManager.Wait()
	if err != nil {
		klog.Error(err)
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

	if entry.Type == irodsfs_common_vpath.VPathVirtualDir {
		// read-only
		return "", nil
	}

	irodsPath, err := entry.GetIRODSPath(vpath)
	if err != nil {
		return "", xerrors.Errorf("failed to get iRODS path for path %q: %w", vpath, err)
	}

	return irodsPath, nil
}

func (syncher *OverlayFSSyncher) isIgnoredFile(path string) bool {
	filename := filepath.Base(path)
	// ignore fuse-overlayfs whiteout files
	// it also uses xattr so we can just ignore them
	if strings.HasPrefix(filename, ".wh.") && strings.HasSuffix(filename, ".opq") {
		return true
	}
	return false
}

func (syncher *OverlayFSSyncher) syncWhiteout(path string, statusFileHandle *irodsclient_fs.FileHandle) error {
	klog.V(5).Infof("processing whiteout file %q", path)

	irodsPath, err := syncher.getIRODSPath(path)
	if err != nil {
		return err
	}

	if len(irodsPath) == 0 {
		klog.V(5).Infof("ignoring %q as it's not writable", path)
		return nil
	}

	if statusFileHandle != nil {
		msg := fmt.Sprintf("Processing whiteout %q\n", irodsPath)
		statusFileHandle.Write([]byte(msg))
	}

	entry, err := syncher.irodsFsClient.Stat(irodsPath)
	if err != nil {
		if irodsclient_types.IsFileNotFoundError(err) {
			// not exist
			klog.Errorf("file or dir %q not exist", irodsPath)
			// suppress warning
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Failed. file or dir %q not exist, ignored\n", irodsPath)
				statusFileHandle.Write([]byte(msg))
			}
			return nil
		}

		if statusFileHandle != nil {
			msg := fmt.Sprintf("> Fail. failed to stat %q. %s\n", irodsPath, err)
			statusFileHandle.Write([]byte(msg))
		}
		return xerrors.Errorf("failed to stat %q: %w", irodsPath, err)
	}

	klog.V(5).Infof("deleting file or dir %q", irodsPath)

	// remove
	if entry.IsDir() {
		err = syncher.irodsFsClient.RemoveDir(irodsPath, true, true)
		if err != nil {
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Fail. failed to remove dir %q. %s\n", irodsPath, err)
				statusFileHandle.Write([]byte(msg))
			}
			return xerrors.Errorf("failed to remove dir %q: %w", irodsPath, err)
		}
	} else {
		err = syncher.irodsFsClient.RemoveFile(irodsPath, true)
		if err != nil {
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Fail. failed to remove file %q. %s\n", irodsPath, err)
				statusFileHandle.Write([]byte(msg))
			}
			return xerrors.Errorf("failed to remove file %q: %w", irodsPath, err)
		}
	}

	if statusFileHandle != nil {
		msg := fmt.Sprintf("> Done. processed whiteout %q\n", irodsPath)
		statusFileHandle.Write([]byte(msg))
	}
	return nil
}

func (syncher *OverlayFSSyncher) syncFile(path string, statusFileHandle *irodsclient_fs.FileHandle) error {
	klog.V(5).Infof("processing new or updated file %q", path)

	irodsPath, err := syncher.getIRODSPath(path)
	if err != nil {
		return err
	}

	if len(irodsPath) == 0 {
		klog.V(5).Infof("ignoring %q as it's not writable", path)
		return nil
	}

	if statusFileHandle != nil {
		msg := fmt.Sprintf("Processing file %q\n", irodsPath)
		statusFileHandle.Write([]byte(msg))
	}

	entry, err := syncher.irodsFsClient.Stat(irodsPath)
	if err != nil {
		if !irodsclient_types.IsFileNotFoundError(err) {
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Fail. failed to stat %q. %s\n", irodsPath, err)
				statusFileHandle.Write([]byte(msg))
			}
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
				if statusFileHandle != nil {
					msg := fmt.Sprintf("> Fail. failed to remove dir %q. %s\n", irodsPath, err)
					statusFileHandle.Write([]byte(msg))
				}
				return xerrors.Errorf("failed to remove dir %q: %w", irodsPath, err)
			}
		}
	}

	klog.V(5).Infof("copying file %q", irodsPath)

	// upload the file
	_, err = syncher.irodsFsClient.UploadFileRedirectToResource(path, irodsPath, "", 0, false, true, true, false, nil)
	if err != nil {
		if statusFileHandle != nil {
			msg := fmt.Sprintf("> Fail. failed to upload file %q. %s\n", irodsPath, err)
			statusFileHandle.Write([]byte(msg))
		}
		return xerrors.Errorf("failed to upload file %q: %w", irodsPath, err)
	}

	if statusFileHandle != nil {
		msg := fmt.Sprintf("> Done. processed file %q\n", irodsPath)
		statusFileHandle.Write([]byte(msg))
	}
	return nil
}

func (syncher *OverlayFSSyncher) syncDir(path string, statusFileHandle *irodsclient_fs.FileHandle) error {
	klog.V(5).Infof("processing dir %q", path)

	irodsPath, err := syncher.getIRODSPath(path)
	if err != nil {
		return err
	}

	if len(irodsPath) == 0 {
		klog.V(5).Infof("ignoring %q as it's not writable", path)
		return nil
	}

	if statusFileHandle != nil {
		msg := fmt.Sprintf("Processing dir %q\n", irodsPath)
		statusFileHandle.Write([]byte(msg))
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
				if statusFileHandle != nil {
					msg := fmt.Sprintf("> Fail. failed to make dir %q. %s\n", irodsPath, err)
					statusFileHandle.Write([]byte(msg))
				}
				return xerrors.Errorf("failed to make dir %q: %w", irodsPath, err)
			}

			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Done. processed dir %q\n", irodsPath)
				statusFileHandle.Write([]byte(msg))
			}
			return nil
		}

		if statusFileHandle != nil {
			msg := fmt.Sprintf("> Fail. failed to stat %q. %s\n", irodsPath, err)
			statusFileHandle.Write([]byte(msg))
		}
		return xerrors.Errorf("failed to stat %q: %w", irodsPath, err)
	}

	// exist
	// if it is a file, remove first
	// if it is a dir, merge or remove
	if !entry.IsDir() {
		// file
		klog.V(5).Infof("deleting file %q", irodsPath)

		err = syncher.irodsFsClient.RemoveFile(irodsPath, true)
		if err != nil {
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Fail. failed to remove file %q. %s\n", irodsPath, err)
				statusFileHandle.Write([]byte(msg))
			}
			return xerrors.Errorf("failed to remove file %q: %w", irodsPath, err)
		}

		klog.V(5).Infof("making dir %q", irodsPath)

		err = syncher.irodsFsClient.MakeDir(irodsPath, true)
		if err != nil {
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Fail. failed to make dir %q. %s\n", irodsPath, err)
				statusFileHandle.Write([]byte(msg))
			}
			return xerrors.Errorf("failed to make dir %q: %w", irodsPath, err)
		}

		if statusFileHandle != nil {
			msg := fmt.Sprintf("> Done. processed dir %q\n", irodsPath)
			statusFileHandle.Write([]byte(msg))
		}
		return nil
	}

	// merge or remove
	if opaqueDir {
		// remove
		klog.V(5).Infof("emptying dir %q", irodsPath)

		err = syncher.clearDirEntries(irodsPath)
		if err != nil {
			if statusFileHandle != nil {
				msg := fmt.Sprintf("> Fail. failed to clear dir %q. %s\n", irodsPath, err)
				statusFileHandle.Write([]byte(msg))
			}
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
