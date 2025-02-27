## iRODS CSI Driver

[iRODS](https://irods.org) Container Storage Interface (CSI) Driver implements the [CSI Specification](https://github.com/container-storage-interface/spec/blob/master/spec.md) to provide container orchestration engines (like [Kubernetes](https://kubernetes.io/)) iRODS access.

### CSI Specification Compatibility

iRODS CSI Driver only supports CSI Specification Version v1.2.0 or higher.

### Features

iRODS CSI Driver relies on external iRODS clients for mounting iRODS collections.
| Driver Type | iRODS Client     | Volume Provisioning | Server Requirements             |
|-------------|------------------|---------------------|---------------------------------|
| irodsfuse   | iRODS FUSE       | Static, Dynamic     | no                              |
| webdav      | DavFS2           | Static              | require [iRODS-WebDAV](https://github.com/DICE-UNC/irods-webdav) or [Davrods](https://github.com/UtrechtUniversity/davrods) |
| nfs         | NFS (nfs-common) | Static              | require [NFS-RODS](https://github.com/irods/irods_client_nfsrods)                |

### Volume Mount Parameters

Parameters specified in Persistent Volume (PV) and Storage Class (SC) are passed to iRODS CSI Driver to mount a volume.
Depending on driver types, different parameters should be given.

For static volume provisioning, parameters are given via Persistent Volume (PV). 
For dynamic volume provisioning, parameters are given via Storage Class (SC).

#### iRODS FUSE Driver
| Field | Description | Example |
| --- | --- | --- |
| client | Driver type | "irodsfuse" |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | "password" in plane text |
| clientuser | iRODS client user id (when using proxy auth) | "irods_cilent_user" |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional, Default "1247" |
| zone | iRODS zone | "iplant" |
| path | iRODS path to mount, starts with **zone** in string | "/iplant/home/irods_user" |
| monitorURL | URL to irodsfs monitor service | "http://monitor.abc.com" |
| pathMappingJSON | JSON string for custom path mappings | "{}" |
| uid | host system UID to map owner | -1 (executor's UID, mostly UID of root, 0) |
| gid | host system GID to map owner | -1 (executor's UID, mostly GID of root, 0) |
| volumeRootPath | iRODS path to mount. Creates a subdirectory per persistent volume. (only for dynamic volume provisioning) | "/iplant/home/irods_user" |
| retainData | "true" to not clear the volume after use. (only for dynamic volume provisioning) | "false". "false" by default. |
| noVolumeDir | "true" to not create a subdirectory under `volumeRootPath`. It mounts the `volumeRootPath`. (only for dynamic volume provisioning) | "false". "false" by default. |
| enforceProxyAccess | "true" to mandate passing `clientUser`, or giving different `user` as in global configuration. | "false". "false" by default. |
| mountPathWhitelist | a comma-separated list of paths to allow mount. | "/iplant/home" |


Mounts **path**

**user** and **password** can be supplied via secrets (nodeStageSecretRef).
Please check out `examples` for more information.

#### WebDAV Driver
| Field | Description | Example |
| --- | --- | --- |
| client | Driver type | "webdav" |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | "password" in plain text |
| url | URL | "https://data.cyverse.org/dav/iplant/home/irods_user" |
| config | additional config parameters in comma-separated kv pairs to be passed to 'davfs2.conf' | "key1=val1,key2=val2" |

Mounts **url**

**user** and **password** can be supplied via secrets (nodeStageSecretRef).
Please check out `examples` for more information.

#### NFS Driver
| Field | Description | Example |
| --- | --- | --- |
| client | Driver type | "nfs" |
| host | WebDAV hostname | "data.cyverse.org" |
| port | WebDAV port | Optional |
| path | iRODS path to mount | "/home/irods_user" |

Mounts **host**:/**path**

### Install & Uninstall

Be aware that the Master branch is not stable! Please use recently released version of code. 

Installation can be done using [Helm Chart Repository](https://cyverse.github.io/irods-csi-driver-helm/), [Helm Chart (manual)](https://github.com/cyverse/irods-csi-driver/tree/master/helm) or by [Manual Deployment](https://github.com/cyverse/irods-csi-driver/tree/master/deploy/kubernetes).

Install using Helm Chart Repository with default configuration:
```shell script
helm repo add irods-csi-driver-repo https://cyverse.github.io/irods-csi-driver-helm/
helm repo update
helm install --create-namespace --namespace irods-csi-driver irods-csi-driver irods-csi-driver-repo/irods-csi-driver
```

Install using Helm Chart with custom configuration:
Edit `helm/user_values.yaml` file. You can set global configuration using the file.

```shell script
helm repo add irods-csi-driver-repo https://cyverse.github.io/irods-csi-driver-helm/
helm repo update
helm install --create-namespace --namespace irods-csi-driver irods-csi-driver irods-csi-driver-repo/irods-csi-driver -f helm/user_values.yaml
```

Uninstall using Helm Chart:
```shell script
helm uninstall --namespace irods-csi-driver irods-csi-driver
```

### Example: Pre-previsioned Persistent Volume (static volume provisioning) using iRODS FUSE

Define Storage Class (SC):
```shell script
kubectl apply -f "examples/kubernetes/static_volume_provisioning/irodsfuse/storageclass.yaml"
```

Define Persistent Volume (PV):
```shell script
kubectl apply -f "examples/kubernetes/static_volume_provisioning/irodsfuse/pv.yaml"
```

Claim Persistent Volume (PVC):
```shell script
kubectl apply -f "examples/kubernetes/static_volume_provisioning/irodsfuse/pvc.yaml"
```

Execute Application with Volume Mount:
```shell script
kubectl apply -f "examples/kubernetes/static_volume_provisioning/irodsfuse/app.yaml"
```

To undeploy, use following command:
```shell script
kubectl delete -f "<YAML file>"
```

Please check out [more examples](https://github.com/cyverse/irods-csi-driver/tree/master/examples).

### References

Following CSI driver implementations were used as references:
- [AWS EFS CSI Driver](https://github.com/kubernetes-sigs/aws-efs-csi-driver)
- [AWS FSx CSI Driver](https://github.com/kubernetes-sigs/aws-fsx-csi-driver)
- [NFS CSI Driver](https://github.com/kubernetes-csi/drivers)
- [Ceph CSI Driver](https://github.com/ceph/ceph-csi)

Many code parts in the driver are from **AWS EFS CSI Driver** and **AWS FSx CSI Driver**.

Following resources are helpful to understand the CSI driver implementation:
- [CSI Specification](https://github.com/container-storage-interface/spec/blob/master/spec.md)
- [Kubernetes CSI Developer Documentation](https://kubernetes-csi.github.io/docs/)

Following resources are helpful to configure the CSI driver:
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)

### License

Copyright (c) 2010-2020, The Arizona Board of Regents on behalf of The University of Arizona

All rights reserved.

Developed by: CyVerse as a collaboration between participants at BIO5 at The University of Arizona (the primary hosting institution), Cold Spring Harbor Laboratory, The University of Texas at Austin, and individual contributors. Find out more at http://www.cyverse.org/.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

 * Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
 * Neither the name of CyVerse, BIO5, The University of Arizona, Cold Spring Harbor Laboratory, The University of Texas at Austin, nor the names of other contributors may be used to endorse or promote products derived from this software without specific prior written permission.


Please check [LICENSE](https://github.com/cyverse/irods-csi-driver/tree/master/LICENSE) file.

#### Code Parts Under Different Licenses

The driver contains open-source code parts under Apache License v2.0.
The code files containing the open-source code parts have the Apache license header in front and which parts are from which code.
Please check [LICENSE.APL2](https://github.com/cyverse/irods-csi-driver/tree/master/LICENSE.APL2) file for the details of Apache License v2.0.
