package volumeinfo

import (
	"encoding/json"
	"os"
	"path"
	"sync"

	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

const (
	controllerVolumeSaveFileName string = "controller_volumes.json"
)

// ControllerVolume class, used by controller to track created volumes
type ControllerVolume struct {
	ID             string                       `yaml:"id" json:"id"`
	Name           string                       `yaml:"name" json:"name"`
	RootPath       string                       `yaml:"root_path" json:"root_path"`
	Path           string                       `yaml:"path" json:"path"`
	ConnectionInfo *irods.IRODSFSConnectionInfo `yaml:"connection_info" json:"connection_info"`
	RetainData     bool                         `yaml:"retain_data" json:"retain_data"`
}

// ControllerVolumeManager manages controller volumes
type ControllerVolumeManager struct {
	encryptKey   string
	savefilePath string
	volumes      map[string]*ControllerVolume
	mutex        sync.Mutex
}

// NewControllerVolumeManager creates ControllerVolumeManager
func NewControllerVolumeManager(encryptKey string, saveDirPath string) (*ControllerVolumeManager, error) {
	if saveDirPath == "" {
		saveDirPath = "/"
	}

	manager := &ControllerVolumeManager{
		encryptKey:   encryptKey,
		savefilePath: path.Join(saveDirPath, controllerVolumeSaveFileName),
		volumes:      map[string]*ControllerVolume{},
		mutex:        sync.Mutex{},
	}

	err := manager.load()
	if err != nil {
		klog.Errorf("failed to access volume file %q, %s. ignoring...", manager.savefilePath, err)
		return manager, nil
	}

	return manager, nil
}

func (manager *ControllerVolumeManager) save() error {
	jsonBytes, err := json.Marshal(manager.volumes)
	if err != nil {
		return status.Errorf(codes.Internal, "json marshal error: %s", err.Error())
	}

	// encrypt data
	if len(manager.encryptKey) > 0 {
		encryptedBytes, err := encrypt(jsonBytes, []byte(manager.encryptKey))
		if err != nil {
			return status.Errorf(codes.Internal, "encrypt error: %s", err.Error())
		}

		err = os.WriteFile(manager.savefilePath, encryptedBytes, 0644)
		if err != nil {
			return status.Errorf(codes.Internal, "write file %q error: %s", manager.savefilePath, err.Error())
		}

		return nil
	}

	// no encryption
	err = os.WriteFile(manager.savefilePath, jsonBytes, 0644)
	if err != nil {
		return status.Errorf(codes.Internal, "write file %q error: %s", manager.savefilePath, err.Error())
	}

	return nil
}

func (manager *ControllerVolumeManager) load() error {
	_, err := os.Stat(manager.savefilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// file not exist
			return nil
		}

		return status.Errorf(codes.Internal, "stat file %q error: %s", manager.savefilePath, err.Error())
	}

	dataBytes, err := os.ReadFile(manager.savefilePath)
	if err != nil {
		return status.Errorf(codes.Internal, "read file %q error: %s", manager.savefilePath, err.Error())
	}

	// decrypt data
	if len(manager.encryptKey) > 0 {
		decryptedBytes, err := decrypt(dataBytes, []byte(manager.encryptKey))
		if err != nil {
			return status.Errorf(codes.Internal, "decrypt error: %s", err.Error())
		}

		if len(decryptedBytes) > 0 && json.Valid(decryptedBytes) {
			// plaintext json
			err = json.Unmarshal(decryptedBytes, &manager.volumes)
			if err != nil {
				return status.Errorf(codes.Internal, "json unmarshal error: %s", err.Error())
			}

			// no error
			return nil
		}

		if len(decryptedBytes) == 0 {
			// empty file
			return nil
		}

		// invalid json
		return status.Errorf(codes.Internal, "invalid json data")
	}

	if len(dataBytes) == 0 {
		// empty file
		return nil
	}

	if len(dataBytes) > 0 && json.Valid(dataBytes) {
		// plaintext json
		err = json.Unmarshal(dataBytes, &manager.volumes)
		if err != nil {
			return status.Errorf(codes.Internal, "json unmarshal error: %s", err.Error())
		}
	}

	return status.Errorf(codes.Internal, "invalid json data")
}

// Get returns the volume with given id
func (manager *ControllerVolumeManager) Get(id string) *ControllerVolume {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	vol, ok := manager.volumes[id]
	if !ok {
		return nil
	}
	return vol
}

// Put puts a volume
func (manager *ControllerVolumeManager) Put(volume *ControllerVolume) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.volumes[volume.ID] = volume

	return manager.save()
}

// Pop returns ControllerVolume with given id and delete
func (manager *ControllerVolumeManager) Pop(id string) (*ControllerVolume, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	vol, ok := manager.volumes[id]
	if ok {
		delete(manager.volumes, id)
		err := manager.save()
		return vol, err
	}
	return nil, nil
}

// Check returns presence of ControllerVolume with given id
func (manager *ControllerVolumeManager) Check(id string) bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	_, ok := manager.volumes[id]
	return ok
}
