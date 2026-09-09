package commons

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeWorkDirsDoesNotRemoveRegularFile(t *testing.T) {
	endpoint := filepath.Join(t.TempDir(), "csi.sock")
	if err := os.WriteFile(endpoint, []byte("do not remove"), 0600); err != nil {
		t.Fatalf("write endpoint file: %v", err)
	}

	config := Config{ServiceEndpoint: "unix://" + endpoint}
	if err := config.MakeWorkDirs(); err == nil {
		t.Fatal("MakeWorkDirs() succeeded for a regular file")
	}

	if _, err := os.Stat(endpoint); err != nil {
		t.Fatalf("endpoint file was removed: %v", err)
	}
}

func TestValidateDriverMode(t *testing.T) {
	controllerConfig := Config{DriverMode: ControllerDriverMode}
	if err := controllerConfig.Validate(); err != nil {
		t.Fatalf("controller config validation failed: %v", err)
	}

	nodeConfig := Config{DriverMode: NodeDriverMode}
	if err := nodeConfig.Validate(); err == nil {
		t.Fatal("node config validation succeeded without a node ID")
	}

	invalidConfig := Config{DriverMode: "invalid"}
	if err := invalidConfig.Validate(); err == nil {
		t.Fatal("invalid driver mode validation succeeded")
	}
}
