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
