package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/cyverse/irods-csi-driver/pkg/client"
	"github.com/cyverse/irods-csi-driver/pkg/client/irods"
	"github.com/cyverse/irods-csi-driver/pkg/commons"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

var (
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}
)

// Dynamic Provisioning: CreateVolume → ControllerPublishVolume (Attach) → NodeStageVolume → NodePublishVolume
// Static Provisioning: ControllerPublishVolume (Attach) → NodeStageVolume → NodePublishVolume

// CreateVolume handles persistent volume creation event
func (driver *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// volume name is created by CO for idempotency
	volName := req.GetName()
	volID := generateVolumeID(volName)

	klog.V(5).Infof("CreateVolume: creating volume %q (%q)", volName, volID)

	if len(volName) == 0 {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if !isValidVolumeCapabilities(volCaps) {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported")
	}

	volCapacity, err := getRequestedCapacity(req.GetCapacityRange())
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	// create a new volume
	// merge params
	configs := commons.MergeConfig(driver.config, driver.secrets, req.GetSecrets(), req.GetParameters())

	///////////////////////////////////////////////////////////
	// We only support irodsfs for dynamic volume provisioning
	///////////////////////////////////////////////////////////
	irodsClientType, err := commons.ParseClientType(configs)
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if irodsClientType != commons.IrodsFuseClientType {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Errorf(codes.InvalidArgument, "unsupported driver type - %v", irodsClientType)
	}

	// make controller config
	controllerConfig, err := parseControllerConfig(volName, configs)
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	// set path
	configs["path"] = controllerConfig.volumePath

	// get iRODS connection info
	irodsConnectionInfo, err := irods.GetConnectionInfo(configs)
	if err != nil {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, err
	}

	// Dynamic provisioning creates a new iRODS directory and therefore must
	// never run with anonymous credentials.
	if irodsConnectionInfo.IsAnonymousUser() {
		commons.IncreaseCounterForVolumeMountFailures()
		return nil, status.Error(codes.InvalidArgument, "dynamic provisioning requires a non-anonymous user")
	}

	// generate path
	if controllerConfig.createVolumeDirectory {
		// create
		klog.V(5).Infof("Creating a volume dir %q", controllerConfig.volumePath)
		err = irods.Mkdir(irodsConnectionInfo, controllerConfig.volumePath)
		if err != nil {
			commons.IncreaseCounterForVolumeMountFailures()
			return nil, status.Errorf(codes.Internal, "Could not create a volume dir %q : %v", controllerConfig.volumePath, err)
		}
	}

	// copy config values to volContext, to be used in node
	volContext := make(map[string]string)
	for k, v := range req.GetParameters() {
		volContext[k] = v
	}
	volContext["path"] = controllerConfig.volumePath

	// tell this volume is created via dynamic volume provisioning
	setDynamicVolumeProvisioningMode(volContext)

	volume := &csi.Volume{
		VolumeId:      volID,
		CapacityBytes: volCapacity,
		VolumeContext: volContext,
	}

	klog.V(5).Infof("CreateVolume: created volume %q (%q)", volName, volID)

	return &csi.CreateVolumeResponse{Volume: volume}, nil
}

// DeleteVolume handles persistent volume deletion event
func (driver *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volID := req.GetVolumeId()

	klog.V(5).Infof("DeleteVolume: deleting volume %q", volID)

	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	klog.V(5).Infof("DeleteVolume: deleted volume %q", volID)

	// Dynamic volume data is retained by policy, so deleting a CSI volume only
	// releases its Kubernetes-side reference.
	return &csi.DeleteVolumeResponse{}, nil
}

func getRequestedCapacity(capacityRange *csi.CapacityRange) (int64, error) {
	if capacityRange == nil {
		return 0, nil
	}

	requiredBytes := capacityRange.GetRequiredBytes()
	limitBytes := capacityRange.GetLimitBytes()
	if requiredBytes < 0 || limitBytes < 0 {
		return 0, status.Error(codes.InvalidArgument, "capacity range values must not be negative")
	}
	if limitBytes > 0 && requiredBytes > limitBytes {
		return 0, status.Error(codes.InvalidArgument, "required_bytes must not exceed limit_bytes")
	}

	return requiredBytes, nil
}

// ControllerPublishVolume handles persistent volume publish event in controller service
func (driver *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerUnpublishVolume handles persistent volume unpublish event in controller service
func (driver *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities returns capabilities
func (driver *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	var caps []*csi.ControllerServiceCapability
	for _, cap := range controllerCaps {
		c := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

// GetCapacity returns volume capacity
func (driver *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes returns a list of volumes created
func (driver *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ValidateVolumeCapabilities checks validity of volume capabilities
func (driver *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	volID := req.GetVolumeId()
	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	confirmed := isValidVolumeCapabilities(volCaps)
	if confirmed {
		if err := driver.validateVolumeConfig(req); err != nil {
			return nil, err
		}

		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				VolumeContext:      req.GetVolumeContext(),
				VolumeCapabilities: volCaps,
				Parameters:         req.GetParameters(),
			},
		}, nil
	}

	return &csi.ValidateVolumeCapabilitiesResponse{}, nil
}

// validateVolumeConfig validates the supplied mount configuration. This
// driver deliberately has no controller-side volume state, so it cannot
// compare VolumeContext with a previously stored value.
func (driver *Driver) validateVolumeConfig(req *csi.ValidateVolumeCapabilitiesRequest) error {
	if len(req.GetVolumeContext()) == 0 && len(req.GetParameters()) == 0 && len(req.GetSecrets()) == 0 {
		return nil
	}

	params := make(map[string]string, len(req.GetParameters())+len(req.GetVolumeContext()))
	for key, value := range req.GetParameters() {
		params[key] = value
	}
	// VolumeContext represents the existing volume and therefore takes
	// precedence over the original CreateVolume parameters.
	for key, value := range req.GetVolumeContext() {
		params[key] = value
	}
	configs := commons.MergeConfig(driver.config, driver.secrets, req.GetSecrets(), params)
	if err := client.ValidateConfig(configs); err != nil {
		return err
	}
	return nil
}

// CreateSnapshot creates a snapshot of a volume
func (driver *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// DeleteSnapshot deletes a snapshot of a volume
func (driver *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListSnapshots returns a list of snapshots
func (driver *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerExpandVolume expands a volume
func (driver *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
