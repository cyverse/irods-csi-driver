package driver

var (
	// key is Id, value is *IRODSVolume
	irodsVolumes = map[string]*IRODSVolume{}
)

// IRODSVolume class
type IRODSVolume struct {
	ID         string
	Name       string
	RootPath   string
	Path       string
	Connection *IRODSConnection
	RetainData bool
}

// NewIRODSVolume returns a new instance of IRODSVolume
func NewIRODSVolume(id string, name string, rootPath string, path string, conn *IRODSConnection, retainData bool) *IRODSVolume {
	return &IRODSVolume{
		ID:         id,
		Name:       name,
		RootPath:   rootPath,
		Path:       path,
		Connection: conn,
		RetainData: retainData,
	}
}

// GetIRODSVolume returns IRODSVolume with given id
func GetIRODSVolume(id string) *IRODSVolume {
	vol, ok := irodsVolumes[id]
	if !ok {
		return nil
	}
	return vol
}

// PutIRODSVolume puts IRODSVolume
func PutIRODSVolume(volume *IRODSVolume) {
	irodsVolumes[volume.ID] = volume
}

// PopIRODSVolume returns IRODSVolume with given id and delete
func PopIRODSVolume(id string) *IRODSVolume {
	vol, ok := irodsVolumes[id]
	if ok {
		delete(irodsVolumes, id)
		return vol
	}
	return nil
}

// CheckIRODSVolume returns presence of IRODSVolume with given id
func CheckIRODSVolume(id string) bool {
	_, ok := irodsVolumes[id]
	return ok
}
