package volumeinfo

import "sync"

// NodeVolume class, used by node to track created volumes
type NodeVolume struct {
	ID                        string
	StagingMountPath          string
	MountPath                 string
	DynamicVolumeProvisioning bool
	StageVolume               bool
}

// NodeVolumeManager manages node volumes
type NodeVolumeManager struct {
	volumes map[string]*NodeVolume
	mutex   sync.Mutex
}

// NewNodeVolumeManager creates ControllerVolumeManager
func NewNodeVolumeManager() *NodeVolumeManager {
	return &NodeVolumeManager{
		volumes: map[string]*NodeVolume{},
		mutex:   sync.Mutex{},
	}
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
func (manager *NodeVolumeManager) Put(volume *NodeVolume) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.volumes[volume.ID] = volume
}

// Pop returns NodeVolume with given id and delete
func (manager *NodeVolumeManager) Pop(id string) *NodeVolume {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	vol, ok := manager.volumes[id]
	if ok {
		delete(manager.volumes, id)
		return vol
	}
	return nil
}

// Check returns presence of NodeVolume with given id
func (manager *NodeVolumeManager) Check(id string) bool {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	_, ok := manager.volumes[id]
	return ok
}
