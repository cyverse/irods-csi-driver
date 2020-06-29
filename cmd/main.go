/*
Copyright 2019 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/cyverse/irods-csi-driver/pkg/driver"
    "github.com/cyverse/irods-csi-driver/pkg/util"

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

    klog.V(1).Infof("Driver version: %s", driver.GetDriverVersion())

    if err := util.ValidateDriverType(conf.DriverType); err != nil {
        klog.Fatalln(err) // calls exit
    }

    klog.V(1).Infof("Starting driver type: %v\n", conf.DriverType)
    drv := driver.NewDriver(&conf)
	if err := drv.Run(); err != nil {
		klog.Fatalln(err)
	}

    os.Exit(0)
}
