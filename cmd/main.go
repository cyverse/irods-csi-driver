package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"syscall"

	"github.com/cyverse/irods-csi-driver/pkg/common"
	"github.com/cyverse/irods-csi-driver/pkg/driver"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/klog"
)

func main() {
	var version bool
	var conf common.Config

	// Parse parameters
	flag.StringVar(&conf.Endpoint, "endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
	flag.StringVar(&conf.NodeID, "nodeid", "", "node id")
	flag.StringVar(&conf.SecretPath, "secretpath", "/etc/irods-csi-dirver", "Secret mount path")
	flag.StringVar(&conf.PoolServiceEndpoint, "poolservice", "unix:///tmp/poolsock", "iRODS FUSE Lite Pool Service endpoint")
	flag.IntVar(&conf.PrometheusExporterPort, "prometheus_exporter_port", 12022, "Prometheus Exporter Service port")
	flag.StringVar(&conf.StoragePath, "storagepath", "/storage", "Storage path for driver internal data")
	flag.BoolVar(&version, "version", false, "Print driver version information")

	klog.InitFlags(nil)
	flag.Parse()

	// Handle Version
	if version {
		info, err := common.GetVersionJSON()
		if err != nil {
			// exit automatically
			klog.Fatalln(err)
		}

		fmt.Println(info)
		os.Exit(0)
	}

	klog.V(1).Infof("Driver version: %s", common.GetDriverVersion())

	if conf.NodeID == "" {
		// exit automatically
		klog.Fatalln("Node ID is not given")
	}

	if conf.StoragePath != "" {
		_, err := os.Stat(conf.StoragePath)
		if err != nil {
			if os.IsNotExist(err) {
				// not exist, make one
				oldMask := syscall.Umask(0)
				defer syscall.Umask(oldMask)

				err = os.MkdirAll(conf.StoragePath, os.FileMode(0777))
				if err != nil {
					klog.Fatalf("Failed to create a storage path %s", conf.StoragePath)
				}
			} else {
				klog.Fatalf("Failed to access a storage path %s", conf.StoragePath)
			}
		}
	}

	// start prometheus exporter server
	var prometheusExporterServer *http.Server
	if conf.PrometheusExporterPort > 0 {
		go func() {
			prometheusExporterAddr := fmt.Sprintf(":%d", conf.PrometheusExporterPort)
			http.Handle("/metrics", promhttp.Handler())

			klog.Infof("Starting prometheus exporter at %s", prometheusExporterAddr)
			prometheusExporterServer = &http.Server{Addr: prometheusExporterAddr, Handler: nil}
			prometheusExporterServer.ListenAndServe()
		}()
	}

	// start driver
	drv, drvErr := driver.NewDriver(&conf)
	if drvErr != nil {
		// shutdown prometheus exporter server when driver fails or stops
		if prometheusExporterServer != nil {
			prometheusExporterServer.Shutdown(context.TODO())
		}

		// exit automatically
		klog.Fatalln(drvErr)
	}

	// driver is created
	err := drv.Run()
	if err != nil {
		// shutdown prometheus exporter server when driver fails or stops
		if prometheusExporterServer != nil {
			prometheusExporterServer.Shutdown(context.TODO())
		}

		// exit automatically
		klog.Fatalln(err)
	}

	// shutdown prometheus exporter server when driver fails or stops
	if prometheusExporterServer != nil {
		prometheusExporterServer.Shutdown(context.TODO())
	}

	os.Exit(0)
}
