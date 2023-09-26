package webdav

import (
	"fmt"
	"os"
	"strings"
)

type DavFSConfig struct {
	Params map[string]string
}

// NewDefaultDavFSConfig creates default DavFS config map
func NewDefaultDavFSConfig() *DavFSConfig {
	return &DavFSConfig{
		Params: map[string]string{},
	}
}

func (config *DavFSConfig) AddParam(key string, value string) {
	config.Params[key] = value
}

func (config *DavFSConfig) AddParams(params map[string]string) {
	for k, v := range params {
		config.Params[k] = v
	}
}

func (config *DavFSConfig) SaveToFile(name string) error {
	sb := strings.Builder{}

	for k, v := range config.Params {
		sb.WriteString(fmt.Sprintf("%s %s\n", k, v))
	}

	return os.WriteFile(name, []byte(sb.String()), 0o664)
}
