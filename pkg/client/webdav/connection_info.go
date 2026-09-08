package webdav

import (
	"net/url"
	"strconv"
	"strings"

	common "github.com/cyverse/irods-csi-driver/pkg/commons"
	irodsfsd_client "github.com/cyverse/irodsfsd/client"
	"github.com/cyverse/irodsfsd/service/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	webdavAnonymousUser string = "anonymous"
)

// WebDAVConnectionInfo class
type WebDAVConnectionInfo struct {
	URL      string
	User     string
	Password string
	ReadOnly bool
	Config   map[string]string
}

// SetAnonymousUser sets anonymous user
func (connInfo *WebDAVConnectionInfo) SetAnonymousUser() {
	connInfo.User = webdavAnonymousUser
}

// IsAnonymousUser checks if the user is anonymous
func (connInfo *WebDAVConnectionInfo) IsAnonymousUser() bool {
	return connInfo.User == webdavAnonymousUser
}

// MakeMountConfig converts the validated WebDAV settings into the daemon API
// configuration. The caller must not log the returned value because it may
// contain a password.
func (connInfo *WebDAVConnectionInfo) MakeMountConfig(targetPath string, mountOptions []string) *irodsfsd_client.MountConfig {
	davfsConfig := &api.DAVFSConfig{
		Url:    connInfo.URL,
		Config: make(map[string]string, len(connInfo.Config)),
	}
	for key, value := range connInfo.Config {
		davfsConfig.Config[key] = value
	}
	if !connInfo.IsAnonymousUser() {
		username := connInfo.User
		password := connInfo.Password
		davfsConfig.Username = &username
		davfsConfig.Password = &password
	}

	return &irodsfsd_client.MountConfig{
		MountPath:    targetPath,
		ReadOnly:     connInfo.ReadOnly || hasReadOnlyOption(mountOptions),
		MountOptions: append([]string(nil), mountOptions...),
		ClientConfig: &api.MountConfig_Davfs{Davfs: davfsConfig},
	}
}

func hasReadOnlyOption(mountOptions []string) bool {
	for _, option := range mountOptions {
		if strings.TrimSpace(option) == "ro" {
			return true
		}
	}
	return false
}

func getConnectionInfoFromMap(params map[string]string, connInfo *WebDAVConnectionInfo) error {
	for k, v := range params {
		switch common.NormalizeConfigKey(k) {
		case common.NormalizeConfigKey("user"), common.NormalizeConfigKey("username"):
			connInfo.User = v
		case common.NormalizeConfigKey("password"), common.NormalizeConfigKey("user_password"):
			connInfo.Password = v
		case common.NormalizeConfigKey("url"):
			connInfo.URL = v
		case common.NormalizeConfigKey("read_only"):
			readOnly, err := strconv.ParseBool(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "argument %q must be a boolean: %v", k, err)
			}
			connInfo.ReadOnly = readOnly
		case common.NormalizeConfigKey("config"):
			config, err := parseDAVFSConfig(v)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "argument %q is invalid: %v", k, err)
			}
			connInfo.Config = config
		default:
			// ignore
		}
	}

	return nil
}

// GetConnectionInfo extracts WebDAVConnectionInfo value from param map
func GetConnectionInfo(configs map[string]string) (*WebDAVConnectionInfo, error) {
	connInfo := WebDAVConnectionInfo{}

	err := getConnectionInfoFromMap(configs, &connInfo)
	if err != nil {
		return nil, err
	}

	// user and password fields are optional
	// if user is not given, it is regarded as anonymous user
	if len(connInfo.User) == 0 {
		connInfo.SetAnonymousUser()
	}

	// password can be empty for anonymous access
	if len(connInfo.Password) == 0 && !connInfo.IsAnonymousUser() {
		return nil, status.Error(codes.InvalidArgument, "Argument password is empty")
	}

	if len(connInfo.URL) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Argument url is empty")
	}

	parsedURL, err := url.ParseRequestURI(connInfo.URL)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL %q", connInfo.URL)
	}
	if parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, status.Errorf(codes.InvalidArgument, "URL %q must be an absolute HTTP(S) URL", connInfo.URL)
	}

	return &connInfo, nil
}

func parseDAVFSConfig(value string) (map[string]string, error) {
	config := make(map[string]string)
	if strings.TrimSpace(value) == "" {
		return config, nil
	}

	for _, entry := range strings.Split(value, ",") {
		keyValue := strings.SplitN(entry, "=", 2)
		if len(keyValue) != 2 {
			return nil, status.Errorf(codes.InvalidArgument, "config entry %q must be key=value", entry)
		}

		key := strings.TrimSpace(keyValue[0])
		configValue := strings.TrimSpace(keyValue[1])
		if key == "" || strings.ContainsAny(key, " \t\r\n") {
			return nil, status.Errorf(codes.InvalidArgument, "config key %q is invalid", key)
		}
		if strings.ContainsAny(configValue, "\r\n") {
			return nil, status.Errorf(codes.InvalidArgument, "config value for %q contains a newline", key)
		}
		config[key] = configValue
	}
	return config, nil
}
