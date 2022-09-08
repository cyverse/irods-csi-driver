package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	promCounterForVolumeMount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irods_csi_driver_volume_mount_total",
		Help: "The total number of irods volumes mounted",
	})
	promCounterForVolumeUnmount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irods_csi_driver_volume_unmount_total",
		Help: "The total number of irods volumes unmounted",
	})
	promCounterForActiveVolumeMount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "irods_csi_driver_volume_mount",
		Help: "The number of current irods volumes",
	})
	promCounterForVolumeMountFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irods_csi_driver_volume_mount_failures_total",
		Help: "The total number of volume mount failures",
	})
	promCounterForVolumeUnmountFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "irods_csi_driver_volume_unmount_failures_total",
		Help: "The total number of volume unmount failures",
	})
)

// IncreaseCounterForVolumeMount increases the counter for volume mount
func IncreaseCounterForVolumeMount() {
	promCounterForVolumeMount.Inc()
}

// IncreaseCounterForVolumeUnmount increases the counter for volume unmount
func IncreaseCounterForVolumeUnmount() {
	promCounterForVolumeUnmount.Inc()
}

// IncreaseCounterForActiveVolumeMount increases the counter for active volume mounts
func IncreaseCounterForActiveVolumeMount() {
	promCounterForActiveVolumeMount.Inc()
}

// DecreaseCounterForActiveVolumeMount decreases the counter for active volume mounts
func DecreaseCounterForActiveVolumeMount() {
	promCounterForActiveVolumeMount.Dec()
}

// IncreaseCounterForVolumeMountFailures increases the counter for volume mount failures
func IncreaseCounterForVolumeMountFailures() {
	promCounterForVolumeMountFailures.Inc()
}

// IncreaseCounterForVolumeUnmountFailures increases the counter for volume unmount failures
func IncreaseCounterForVolumeUnmountFailures() {
	promCounterForVolumeUnmountFailures.Inc()
}
