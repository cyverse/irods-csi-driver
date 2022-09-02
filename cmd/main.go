package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

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
	flag.BoolVar(&version, "version", false, "Print driver version information")

	klog.InitFlags(nil)
	flag.Parse()

	// Handle Version
	if version {
		info, err := common.GetVersionJSON()

		if err != nil {
			klog.Fatalln(err)
		}

		fmt.Println(info)
		os.Exit(0)
	}

	klog.V(1).Infof("Driver version: %s", common.GetDriverVersion())

	if conf.NodeID == "" {
		klog.Fatalln("Node ID is not given")
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
	drv := driver.NewDriver(&conf)
	if err := drv.Run(); err != nil {
		klog.Fatalln(err)
	}

	// shutdown prometheus exporter server when driver fails or stops
	if prometheusExporterServer != nil {
		prometheusExporterServer.Shutdown(context.TODO())
	}

	os.Exit(0)
}
