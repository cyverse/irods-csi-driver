package volumeinfo

import (
	"encoding/json"
	"os"
	"path"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
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
	ClientType                string            `yaml:"client_type" json:"client_type"`
	ClientConfig              map[string]string `yaml:"client_config" json:"client_config"`
	DynamicVolumeProvisioning bool              `yaml:"dynamic_volume_provisioning" json:"dynamic_volume_provisioning"`
	StageVolume               bool              `yaml:"stage_volume" json:"stage_volume"`
}

// NodeVolumeManager manages node volumes
type NodeVolumeManager struct {
	encryptKey   string
	savefilePath string
	volumes      map[string]*NodeVolume
	mutex        sync.Mutex
}

// NewNodeVolumeManager creates ControllerVolumeManager
func NewNodeVolumeManager(encryptKey string, saveDirPath string) (*NodeVolumeManager, error) {
	if saveDirPath == "" {
		saveDirPath = "/"
	}

	manager := &NodeVolumeManager{
		encryptKey:   encryptKey,
		savefilePath: path.Join(saveDirPath, nodeVolumeSaveFileName),
		volumes:      map[string]*NodeVolume{},
		mutex:        sync.Mutex{},
	}

	err := manager.load()
	if err != nil {
		klog.Errorf("failed to access volume file %q, %s. ignoring...", manager.savefilePath, err)
		return manager, nil
	}

	return manager, nil
}

func (manager *NodeVolumeManager) save() error {
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

func (manager *NodeVolumeManager) load() error {
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
