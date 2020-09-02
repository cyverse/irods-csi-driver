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
| driver (or client) | Driver type | "irodsfuse" |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | "password" in plane text |
| clientuser | iRODS client user id (when using proxy auth) | "irods_cilent_user" |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional, Default "1247" |
| ticket | Ticket string | Optional |
| zone | iRODS zone | "iplant" |
| path | iRODS path to mount, does not include **zone** in string | "/home/irods_user" |

Mounts **zone**/**path**

**user**, **password** and **ticket** can be supplied via secrets (nodePublishSecretRef).
Please check out `examples` for more information.

#### WebDAV Driver
| Field | Description | Example |
| --- | --- | --- |
| driver (or client) | Driver type | "webdav" |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | "password" in plane text |
| url | URL | "https://data.cyverse.org/dav/iplant/home/irods_user" |

Mounts **url**

**user** and **password** can be supplied via secrets (nodePublishSecretRef).
Please check out `examples` for more information.

#### NFS Driver
| Field | Description | Example |
| --- | --- | --- |
| driver (or client) | Driver type | "nfs" |
| host | WebDAV hostname | "data.cyverse.org" |
| port | WebDAV port | Optional |
| path | iRODS path to mount | "/home/irods_user" |

Mounts **host**:/**path**

### Installation

Deploy the stable driver:

```sh
kubectl apply -k "deploy/kubernetes/overlays/stable"
```

Deploy the development driver:
```sh
kubectl apply -k "deploy/kubernetes/overlays/dev"
```

Verify the driver installation:
```sh
kubectl get csinodes -o jsonpath='{range .items[*]} {.metadata.name}{": "} {range .spec.drivers[*]} {.name}{"\n"} {end}{end}'
```

### Uninstallation

Uninstall the stable driver:
```sh
kubectl delete -k "deploy/kubernetes/overlays/stable"
```

Uninstall the development driver:
```sh
kubectl delete -k "deploy/kubernetes/overlays/dev"
```

### Mount Pre-previsioned Persistent Volume using iRODS FUSE

Define Storage Class (SC):
```sh
kubectl apply -f "examples/kubernetes/irodsfuse_static_provisioning/storageclass.yaml"
```

Verify the Storage Class definition:
```sh
kubectl get sc
```

Define Persistent Volume (PV):
```sh
kubectl apply -f "examples/kubernetes/irodsfuse_static_provisioning/pv.yaml"
```

Verify the Persistent Volume definition:
```sh
kubectl get pv
```

Claim Persistent Volume (PVC):
```sh
kubectl apply -f "examples/kubernetes/irodsfuse_static_provisioning/pvc.yaml"
```

Verify the Persistent Volume Claim:
```sh
kubectl get pvc
```

Execute Application with Volume Mount:
```sh
kubectl apply -f "examples/kubernetes/irodsfuse_static_provisioning/app.yaml"
```

### Unmount  Pre-previsioned Persistent Volume using iRODS FUSE

Delete Application:
```sh
kubectl delete --grace-period=0 --force -f "examples/kubernetes/irodsfuse_static_provisioning/app.yaml"
```

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

This code is licensed under the Apache 2.0 License.
