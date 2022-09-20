package volumeinfo

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	savefilePath string
	volumes      map[string]*ControllerVolume
	mutex        sync.Mutex
}

// NewControllerVolumeManager creates ControllerVolumeManager
func NewControllerVolumeManager(saveDirPath string) (*ControllerVolumeManager, error) {
	if saveDirPath == "" {
		saveDirPath = "/"
	}

	manager := &ControllerVolumeManager{
		savefilePath: path.Join(saveDirPath, controllerVolumeSaveFileName),
		volumes:      map[string]*ControllerVolume{},
		mutex:        sync.Mutex{},
	}

	err := manager.load()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func (manager *ControllerVolumeManager) save() error {
	jsonBytes, err := json.Marshal(manager.volumes)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	return ioutil.WriteFile(manager.savefilePath, jsonBytes, 0644)
}

func (manager *ControllerVolumeManager) load() error {
	_, err := os.Stat(manager.savefilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// file not exist
			return nil
		}
		return status.Errorf(codes.Internal, err.Error())
	}

	jsonBytes, err := ioutil.ReadFile(manager.savefilePath)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	return json.Unmarshal(jsonBytes, &manager.volumes)
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
