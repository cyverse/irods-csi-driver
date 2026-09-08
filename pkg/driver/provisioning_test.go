package driver

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGenerateVolumeIDIsDeterministic(t *testing.T) {
	volumeName := "pvc:123"
	first := generateVolumeID(volumeName)
	second := generateVolumeID(volumeName)
	if first != second {
		t.Fatalf("same volume name generated different IDs: %q and %q", first, second)
	}
	if first == generateVolumeID("pvc-456") {
		t.Fatal("different volume names generated the same ID")
	}
	if decodedName, err := volumeNameFromID(first); err != nil || decodedName != volumeName {
		t.Fatalf("volumeNameFromID(%q) = %q, %v; want %q, nil", first, decodedName, err, volumeName)
	}
}

func TestParseControllerConfig(t *testing.T) {
	tests := []struct {
		name                string
		volumeName          string
		params              map[string]string
		wantRoot            string
		wantPath            string
		wantCreateVolumeDir bool
		wantErrorCode       codes.Code
	}{
		{
			name:                "creates volume directory below cleaned root",
			volumeName:          "pvc-123",
			params:              map[string]string{"volume_root_path": "/tempZone/home/alice/"},
			wantRoot:            "/tempZone/home/alice",
			wantPath:            "/tempZone/home/alice/pvc-123",
			wantCreateVolumeDir: true,
		},
		{
			name:                "uses root without creating a directory",
			volumeName:          "pvc-123",
			params:              map[string]string{"volume_root_path": "/tempZone/home/alice", "no_volume_dir": "true"},
			wantRoot:            "/tempZone/home/alice",
			wantPath:            "/tempZone/home/alice",
			wantCreateVolumeDir: false,
		},
		{
			name:          "rejects relative root",
			volumeName:    "pvc-123",
			params:        map[string]string{"volume_root_path": "relative"},
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:          "rejects path traversal volume name",
			volumeName:    "../other",
			params:        map[string]string{"volume_root_path": "/tempZone/home/alice"},
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:          "rejects invalid no volume directory value",
			volumeName:    "pvc-123",
			params:        map[string]string{"volume_root_path": "/tempZone/home/alice", "no_volume_dir": "sometimes"},
			wantErrorCode: codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := parseControllerConfig(test.volumeName, test.params)
			if test.wantErrorCode != codes.OK {
				if status.Code(err) != test.wantErrorCode {
					t.Fatalf("parseControllerConfig() error = %v, want code %s", err, test.wantErrorCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if config.volumeRootPath != test.wantRoot || config.volumePath != test.wantPath ||
				config.createVolumeDirectory != test.wantCreateVolumeDir {
				t.Fatalf("config = %#v", config)
			}
		})
	}
}

func TestDynamicVolumeProvisioningMode(t *testing.T) {
	if !isDynamicVolumeProvisioningMode(map[string]string{"provisioning_mode": " DYNAMIC "}) {
		t.Fatal("dynamic provisioning mode was not recognized")
	}
	if isDynamicVolumeProvisioningMode(map[string]string{"provisioning_mode": "static"}) {
		t.Fatal("static provisioning mode was recognized as dynamic")
	}
}

func TestGetRequestedCapacity(t *testing.T) {
	tests := []struct {
		name          string
		capacityRange *csi.CapacityRange
		wantCapacity  int64
		wantErrorCode codes.Code
	}{
		{name: "unspecified", wantCapacity: 0},
		{name: "required bytes", capacityRange: &csi.CapacityRange{RequiredBytes: 1024}, wantCapacity: 1024},
		{name: "required bytes exceeds limit", capacityRange: &csi.CapacityRange{RequiredBytes: 1024, LimitBytes: 512}, wantErrorCode: codes.InvalidArgument},
		{name: "negative required bytes", capacityRange: &csi.CapacityRange{RequiredBytes: -1}, wantErrorCode: codes.InvalidArgument},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capacity, err := getRequestedCapacity(test.capacityRange)
			if test.wantErrorCode != codes.OK {
				if status.Code(err) != test.wantErrorCode {
					t.Fatalf("getRequestedCapacity() error = %v, want code %s", err, test.wantErrorCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if capacity != test.wantCapacity {
				t.Fatalf("getRequestedCapacity() = %d, want %d", capacity, test.wantCapacity)
			}
		})
	}
}
