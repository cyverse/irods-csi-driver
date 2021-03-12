## Static Volume Provisioning

In static volume provisioning, persistent volumes must be pre-provisioned before they are claimed. Static volume provisioning examples includes "pv.yaml" files to pre-provision the volumes.

## iRODS Configuration

The "pv.yaml" files contain iRODS information including username, password and iRODS host information.

### iRODS Client Configuration

Following iRODS clients can be used for the static volume provisioning.
| Driver Type | iRODS Client     | Server Requirements             |
|-------------|------------------|---------------------------------|
| irodsfuse   | iRODS FUSE       | no                              |
| webdav      | DavFS2           | require [iRODS-WebDAV](https://github.com/DICE-UNC/irods-webdav) or [Davrods](https://github.com/UtrechtUniversity/davrods) |
| nfs         | NFS (nfs-common) | require [NFS-RODS](https://github.com/irods/irods_client_nfsrods)                |

#### iRODS FUSE Client
| Field | Description | Example |
| --- | --- | --- |
| client (or driver) | Client type | "irodsfuse" |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | "password" in plane text |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional, Default "1247" |
| zone | iRODS zone | "iplant" |
| path | iRODS path to mount, does not include **zone** in string | "/home/irods_user" |

Mounts **zone**/**path**

#### WebDAV Client
| Field | Description | Example |
| --- | --- | --- |
| client (or driver) | Client type | "webdav" |
| user | iRODS user id | "irods_user" or leave empty for anonymous access |
| password | iRODS user password | "password" in plane text or leave empty for anonymous access |
| url | URL | "https://data.cyverse.org/dav/iplant/home/irods_user" |

Mounts **url**

#### NFS Client
| Field | Description | Example |
| --- | --- | --- |
| client (or driver) | Driver type | "nfs" |
| host | WebDAV hostname | "data.cyverse.org" |
| port | WebDAV port | Optional |
| path | iRODS path to mount | "/home/irods_user" |

Mounts **host**:/**path**

### Kubernetes Secrets

Optionally, Kubernetes Secrets can be used to pass sensitive informations such as username and password. iRODS host information also can be passed in this way.
Kubernetes Secrets can be supplied via **nodeStageSecretRef**.

### Execute examples in following order

Define Storage Class (SC):
```shell script
kubectl apply -f "irodsfuse/storageclass.yaml"
```

Define Persistent Volume (PV):
```shell script
kubectl apply -f "irodsfuse/pv.yaml"
```

Claim Persistent Volume (PVC):
```shell script
kubectl apply -f "irodsfuse/pvc.yaml"
```

Execute Application with Volume Mount:
```shell script
kubectl apply -f "irodsfuse/app.yaml"
```

Undeployment must be done in reverse order.
```shell script
kubectl delete -f "<YAML file>"
```