package nfs

import "testing"

func TestGetConnectionInfo(t *testing.T) {
	connectionInfo, err := GetConnectionInfo(map[string]string{
		"host":     "nfs.example.org",
		"path":     "/exports/data",
		"readOnly": "true",
	})
	if err != nil {
		t.Fatalf("GetConnectionInfo() error = %v", err)
	}
	if connectionInfo.Port != 2049 {
		t.Errorf("Port = %d, want 2049", connectionInfo.Port)
	}
	if !connectionInfo.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
}

func TestGetConnectionInfoRejectsInvalidInput(t *testing.T) {
	testCases := []map[string]string{
		{"path": "/exports/data"},
		{"host": "nfs.example.org", "path": "exports/data"},
		{"host": "nfs.example.org", "path": "/exports/data", "port": "-1"},
		{"host": "nfs.example.org", "path": "/exports/data", "port": "65536"},
		{"host": "nfs.example.org", "path": "/exports/data", "readOnly": "invalid"},
	}
	for _, configs := range testCases {
		if _, err := GetConnectionInfo(configs); err == nil {
			t.Errorf("GetConnectionInfo(%#v) succeeded, want error", configs)
		}
	}
}

func TestMakeMountConfig(t *testing.T) {
	connectionInfo := &NFSConnectionInfo{Hostname: "nfs.example.org", Port: 2050, Path: "/exports/data", ReadOnly: true}
	config := connectionInfo.MakeMountConfig("/target", []string{"rw"})
	if !config.GetReadOnly() {
		t.Error("ReadOnly = false, want true")
	}
	if config.GetNfs().GetHost() != connectionInfo.Hostname || config.GetNfs().GetPort() != int32(connectionInfo.Port) || config.GetNfs().GetPath() != connectionInfo.Path {
		t.Errorf("NFS config = %#v", config.GetNfs())
	}
}
