## Dynamic Volume Provisioning

In dynamic volume provisioning, persistent volumes are automatically provisioned by the CSI driver when they are claimed. Dynamic volume provisioning examples includes "storageclass.yaml" files to define parameters that are used in persistent volume provisioning.

## iRODS Configuration

The "storageclass.yaml" files contain iRODS information including username, password and iRODS host information.

### iRODS Client Configuration

Following iRODS clients can be used for the dynamic volume provisioning.
| Driver Type | iRODS Client     | Server Requirements             |
|-------------|------------------|---------------------------------|
| irodsfuse   | iRODS FUSE       | no                              |

#### iRODS FUSE Client
| Field | Description | Example |
| --- | --- | --- |
| client (or driver) | Client type | "irodsfuse". "irodsfuse" by default. |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | "password" in plane text |
| clientUser | iRODS client user id (when using proxy auth) | "irods_client_user" or leave empty |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional. "1247" by default. |
| zone | iRODS zone | "iplant" |
| volumeRootPath | iRODS path to mount. Creates a subdirectory per persistent volume. Path starts with **zone** in string | "/iplant/home/irods_user" |
| retainData | "true" to not clear the volume after use. | "false". "false" by default. |
| noVolumeDir | "true" to not create a subdirectory under `volumeRootPath`. It mounts the `volumeRootPath`. | "false". "false" by default. |
| enforceProxyAccess | "true" to mandate passing `clientUser`, or giving different `user` as in global configuration. | "false". "false" by default. |

Mounts **path**

### Kubernetes Secrets

Optionally, Kubernetes Secrets can be used to pass sensitive informations such as username and password. iRODS host information also can be passed in this way.
Kubernetes Secrets can be supplied via **csiProvisionerSecretName**, **csiProvisionerSecretNamespace**, **csiNodeStageSecretName** and **csiNodeStageSecretNamespace**.

### Execute examples in following order

Define Storage Class (SC):
```shell script
kubectl apply -f "irodsfuse/storageclass.yaml"
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