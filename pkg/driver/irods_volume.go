/*
Copyright 2020 CyVerse
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
