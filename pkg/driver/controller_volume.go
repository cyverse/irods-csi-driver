package driver

// ControllerVolume class
type ControllerVolume struct {
	ID             string
	Name           string
	RootPath       string
	Path           string
	ConnectionInfo *IRODSConnectionInfo
	RetainData     bool
}

// GetControllerVolume returns ControllerVolume with given id
func (driver *Driver) GetControllerVolume(id string) *ControllerVolume {
	driver.volumeLock.Lock()
	defer driver.volumeLock.Unlock()

	vol, ok := driver.controllerVolumes[id]
	if !ok {
		return nil
	}
	return vol
}

// PutControllerVolume puts ControllerVolume
func (driver *Driver) PutControllerVolume(volume *ControllerVolume) {
	driver.volumeLock.Lock()
	defer driver.volumeLock.Unlock()

	driver.controllerVolumes[volume.ID] = volume
}

// PopControllerVolume returns ControllerVolume with given id and delete
func (driver *Driver) PopControllerVolume(id string) *ControllerVolume {
	driver.volumeLock.Lock()
	defer driver.volumeLock.Unlock()

	vol, ok := driver.controllerVolumes[id]
	if ok {
		delete(driver.controllerVolumes, id)
		return vol
	}
	return nil
}

// CheckControllerVolume returns presence of ControllerVolume with given id
func (driver *Driver) CheckControllerVolume(id string) bool {
	driver.volumeLock.Lock()
	defer driver.volumeLock.Unlock()

	_, ok := driver.controllerVolumes[id]
	return ok
}
