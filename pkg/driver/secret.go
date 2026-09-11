package driver

import (
	"os"
	"path"
	"strings"

	"github.com/cockroachdb/errors"
)

// readSecrets reads secrets from secret volume mount
func readSecrets(secretPath string) (map[string]string, error) {
	exist, err := PathExists(secretPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Secret path %q does not exist", secretPath)
	}

	if !exist {
		return nil, errors.Newf("Secret path %q does not exist", secretPath)
	}

	secrets := make(map[string]string)

	files, err := os.ReadDir(secretPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			k := path.Base(file.Name())
			fullPath := path.Join(secretPath, k)
			content, readErr := os.ReadFile(fullPath)
			if readErr == nil {
				contentString := string(content)
				v := strings.TrimSpace(contentString)
				secrets[k] = v
			}
		}
	}
	return secrets, nil
}
