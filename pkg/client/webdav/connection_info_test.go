package webdav

import "testing"

func TestMakeMountConfig(t *testing.T) {
	connectionInfo := &WebDAVConnectionInfo{
		URL:      "https://webdav.example.org/data",
		User:     "user",
		Password: "secret",
		Config:   map[string]string{"use_locks": "0"},
	}

	config := connectionInfo.MakeMountConfig("/var/lib/kubelet/pods/target", []string{"rw", "ro"})
	if config.GetMountPath() != "/var/lib/kubelet/pods/target" {
		t.Errorf("mount path = %q", config.GetMountPath())
	}
	if !config.GetReadOnly() {
		t.Error("ReadOnly = false, want true")
	}
	if config.GetDavfs().GetUrl() != connectionInfo.URL {
		t.Errorf("URL = %q", config.GetDavfs().GetUrl())
	}
	if config.GetDavfs().GetUsername() != connectionInfo.User {
		t.Errorf("username = %q", config.GetDavfs().GetUsername())
	}
	if config.GetDavfs().GetPassword() != connectionInfo.Password {
		t.Errorf("password = %q", config.GetDavfs().GetPassword())
	}
	if config.GetDavfs().GetConfig()["use_locks"] != "0" {
		t.Errorf("config = %#v", config.GetDavfs().GetConfig())
	}
}

func TestMakeMountConfigForAnonymousUserOmitsCredentials(t *testing.T) {
	connectionInfo := &WebDAVConnectionInfo{URL: "https://webdav.example.org/data", User: webdavAnonymousUser}
	config := connectionInfo.MakeMountConfig("/target", nil)
	if config.GetDavfs().Username != nil || config.GetDavfs().Password != nil {
		t.Error("anonymous config must not include credentials")
	}
}

func TestGetConnectionInfoParsesReadOnlyAndDAVFSConfig(t *testing.T) {
	connectionInfo, err := GetConnectionInfo(map[string]string{
		"url":      "https://webdav.example.org/data",
		"user":     "user",
		"password": "password",
		"readOnly": "true",
		"config":   "use_locks=0,token=a=b",
	})
	if err != nil {
		t.Fatalf("GetConnectionInfo() error = %v", err)
	}
	if !connectionInfo.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
	if connectionInfo.Config["token"] != "a=b" {
		t.Errorf("config token = %q, want %q", connectionInfo.Config["token"], "a=b")
	}
}

func TestGetConnectionInfoRejectsInvalidInput(t *testing.T) {
	testCases := []map[string]string{
		{"url": "relative/path"},
		{"url": "ftp://webdav.example.org/data"},
		{"url": "https://webdav.example.org/data", "readOnly": "invalid"},
		{"url": "https://webdav.example.org/data", "config": "missing-value"},
		{"url": "https://webdav.example.org/data", "config": "bad key=value"},
	}
	for _, configs := range testCases {
		if _, err := GetConnectionInfo(configs); err == nil {
			t.Errorf("GetConnectionInfo(%#v) succeeded, want error", configs)
		}
	}
}
