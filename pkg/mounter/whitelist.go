package mounter

import (
	"path/filepath"
	"strings"
)

func isSubDir(parent string, sub string) bool {
	rel, err := filepath.Rel(parent, sub)
	if err != nil {
		return false
	}

	if !strings.HasPrefix(rel, "..") {
		return true
	}
	return false
}

// IsMountPathAllowed checks if given path is allowed to mount
func IsMountPathAllowed(whitelist []string, path string) bool {
	for _, item := range whitelist {
		if isSubDir(item, path) {
			return true
		}
	}

	return false
}
