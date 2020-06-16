package main

import (
    "flag"
    "fmt"
    "os"
    "runtime"

    "github.com/iychoi/irods-csi-driver/pkg/driver"
    "github.com/iychoi/irods-csi-driver/pkg/util"

    "k8s.io/klog"
)

var (
	conf util.Config
)

func main() {
    // Parse parameters
    flag.StringVar(&conf.DriverType, "type", "", "driver type [fuse|nfs|webdav]")
    flag.StringVar(&conf.Endpoint, "endpoint", "unix://tmp/csi.sock", "CSI endpoint")

    flag.BoolVar(&conf.Version, "version", false, "Print driver version information")


    klog.InitFlags(nil)
	flag.Parse()

    // Handle Version
    if conf.Version {
        info, err := driver.GetVersionJSON()

        if err != nil {
            klog.Fatalln(err)
        }

        fmt.Println(info)
        os.Exit(0)
    }

    klog.V(1).Infof("Driver version: %s", driver.driverVersion)

    err := util.ValidateDriverType(conf.DriverType)
    if err != nil {
        klog.Fatalln(err) // calls exit
    }

    klog.V(1).Infof("Starting driver type: %v\n", conf.DriverType)
    drv := driver.NewDriver(&conf)
    err := drv.Run()
	if err != nil {
		klog.Fatalln(err)
	}

    os.Exit(0)
}
