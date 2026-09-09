package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/cyverse/irods-csi-driver/pkg/commons"
	"github.com/cyverse/irods-csi-driver/pkg/driver"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/klog"
)

func main() {
	var version bool
	var conf commons.Config

	// Parse parameters
	flag.StringVar(&conf.ServiceEndpoint, "endpoint", "", "CSI endpoint")
	flag.StringVar((*string)(&conf.DriverMode), "mode", "", "CSI driver mode: controller or node")
	flag.StringVar(&conf.NodeID, "nodeid", "", "node id")
	flag.StringVar(&conf.SecretPath, "secretpath", "/etc/irods-csi-dirver", "Secret mount path")
	flag.StringVar(&conf.IRODSFSDServiceEndpoint, "irodsfsd-endpoint", "tcp://127.0.0.1:13020", "iRODS FSD service endpoint")
	flag.IntVar(&conf.PrometheusExporterPort, "prometheus_exporter_port", 14021, "Prometheus Exporter Service port")
	flag.BoolVar(&version, "version", false, "Print driver version information")

	klog.InitFlags(nil)
	flag.Parse()

	// Handle Version
	if version {
		info, err := commons.GetVersionJSON()
		if err != nil {
			// exit automatically
			klog.Fatal(err)
		}

		fmt.Println(info)
		return
	}

	klog.V(1).Infof("Driver version: %q", commons.GetDriverVersion())

	err := conf.Validate()
	if err != nil {
		// exit automatically
		klog.Fatal(err)
	}

	err = conf.MakeWorkDirs()
	if err != nil {
		// exit automatically
		klog.Fatal(err)
	}

	// start prometheus exporter server
	var prometheusExporterServer *http.Server
	if conf.PrometheusExporterPort > 0 {
		go func() {
			prometheusExporterAddr := fmt.Sprintf(":%d", conf.PrometheusExporterPort)
			http.Handle("/metrics", promhttp.Handler())

			klog.Infof("Starting prometheus exporter at %q", prometheusExporterAddr)
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
		klog.Fatal(drvErr)
	}

	// driver is created
	drvErr = drv.Run()
	if drvErr != nil {
		// shutdown prometheus exporter server when driver fails or stops
		if prometheusExporterServer != nil {
			prometheusExporterServer.Shutdown(context.TODO())
		}

		// exit automatically
		klog.Fatal(drvErr)
	}

	// shutdown prometheus exporter server when driver fails or stops
	if prometheusExporterServer != nil {
		prometheusExporterServer.Shutdown(context.TODO())
	}

	return
}
