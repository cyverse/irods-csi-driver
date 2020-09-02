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

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"
)

// ReadIRODSSecrets reads secrets from secret volume mount
func ReadIRODSSecrets(secretPath string) (map[string]string, error) {
	exist, err := PathExists(secretPath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Secret path %s does not exist - %s", secretPath, err))
	}

	if !exist {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Secret path %s does not exist", secretPath))
	}

	secrets := make(map[string]string)

	files, err := ioutil.ReadDir(secretPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			k := path.Base(file.Name())
			fullPath := path.Join(secretPath, k)
			content, readErr := ioutil.ReadFile(fullPath)
			if readErr == nil {
				contentString := string(content)
				v := strings.TrimSpace(contentString)
				secrets[k] = v
			}
		}
	}
	return secrets, nil
}
