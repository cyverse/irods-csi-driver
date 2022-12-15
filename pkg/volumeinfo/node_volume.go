package volumeinfo

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeVolumeStatus string

const (
	nodeVolumeSaveFileName string = "node_volumes.json"

	NodeVolumeStatusStage string = "stage"
)

// NodeVolume class, used by node to track created volumes
type NodeVolume struct {
	ID                        string            `yaml:"id" json:"id"`
	StagingMountPath          string            `yaml:"staging_mount_path" json:"staging_mount_path"`
	MountPath                 string            `yaml:"mount_path" json:"mount_path"`
	StagingMountOptions       []string          `yaml:"staging_mount_options" json:"staging_mount_options"`
	MountOptions              []string          `yaml:"mount_options" json:"mount_options"`
	ClientConfig              map[string]string `yaml:"client_config" json:"client_config"`
	DynamicVolumeProvisioning bool              `yaml:"dynamic_volume_provisioning" json:"dynamic_volume_provisioning"`
	StageVolume               bool              `yaml:"stage_volume" json:"stage_volume"`
}

// NodeVolumeManager manages node volumes
type NodeVolumeManager struct {
	savefilePath string
	volumes      map[string]*NodeVolume
	mutex        sync.Mutex
}

// NewNodeVolumeManager creates ControllerVolumeManager
func NewNodeVolumeManager(saveDirPath string) (*NodeVolumeManager, error) {
	if saveDirPath == "" {
		saveDirPath = "/"
	}

	manager := &NodeVolumeManager{
		savefilePath: path.Join(saveDirPath, nodeVolumeSaveFileName),
		volumes:      map[string]*NodeVolume{},
		mutex:        sync.Mutex{},
	}

	err := manager.load()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func (manager *NodeVolumeManager) save() error {
	jsonBytes, err := json.Marshal(manager.volumes)
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}

	return ioutil.WriteFile(manager.savefilePath, jsonBytes, 0644)
}

func (manager *NodeVolumeManager) load() error {
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
func (manager *NodeVolumeManager) Get(id string) *NodeVolume {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	vol, ok := manager.volumes[id]
	if !ok {
		return nil
	}
	return vol
}

// Put puts a volume
func (manager *NodeVolumeManager) Put(volume *NodeVolume) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.volumes[volume.ID] = volume

	return manager.save()
}

// Pop returns NodeVolume with given id and delete
func (manager *NodeVolumeManager) Pop(id string) (*NodeVolume, error) {
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

// Check returns presence of NodeVolume with given id
func (manager *NodeVolumeManager) Check(id string) bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	_, ok := manager.volumes[id]
	return ok
}
