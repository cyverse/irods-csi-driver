package volumeinfo

import (
	"sync"

	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
)

// ControllerVolume class, used by controller to track created volumes
type ControllerVolume struct {
	ID             string
	Name           string
	RootPath       string
	Path           string
	ConnectionInfo *irods.IRODSFSConnectionInfo
	RetainData     bool
}

// ControllerVolumeManager manages controller volumes
type ControllerVolumeManager struct {
	volumes map[string]*ControllerVolume
	mutex   sync.Mutex
}

// NewControllerVolumeManager creates ControllerVolumeManager
func NewControllerVolumeManager() *ControllerVolumeManager {
	return &ControllerVolumeManager{
		volumes: map[string]*ControllerVolume{},
		mutex:   sync.Mutex{},
	}
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
func (manager *ControllerVolumeManager) Put(volume *ControllerVolume) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.volumes[volume.ID] = volume
}

// Pop returns ControllerVolume with given id and delete
func (manager *ControllerVolumeManager) Pop(id string) *ControllerVolume {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	vol, ok := manager.volumes[id]
	if ok {
		delete(manager.volumes, id)
		return vol
	}
	return nil
}

// Check returns presence of ControllerVolume with given id
func (manager *ControllerVolumeManager) Check(id string) bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	_, ok := manager.volumes[id]
	return ok
}
