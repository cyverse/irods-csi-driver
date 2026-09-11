## iRODS CSI Driver Helm Chart
This chart enables easy installation of the iRODS CSI Driver using Helm.

### Compatibility
- Helm 3+
- Kubernetes > 1.17.x, can be deployed to any namespace.
- Kubernetes < 1.17.x, namespace **must** be `kube-system`, as `system-cluster-critical` is hard-coded to this namespace.

### Prerequisites

`irodsfs` and `irodsfsd` must be installed before installing or using the iRODS CSI Driver. You can obtain them from the following repositories:

- [irodsfs](https://github.com/cyverse/irodsfs)
- [irodsfsd](https://github.com/cyverse/irodsfsd)

### Global iRODS FUSE Configuration

Set global iRODS FUSE parameters under `globalConfig.secret.stringData` in a
Helm values file, such as `user_values.yaml`. The chart creates a Kubernetes
Secret from these values and mounts it in both CSI driver components. Global
values override PV or StorageClass parameters and `nodeStageSecretRef` values.
All values must be strings.

| Field | Description | Example |
| --- | --- | --- |
| client | Driver type | "irodsfuse" |
| authenticationScheme | iRODS authentication scheme | "native" |
| clientServerNegotiation | iRODS client-server negotiation setting | "request_server_negotiation" |
| clientServerPolicy | iRODS client-server negotiation policy | "CS_NEG_REQUIRE" |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional, Default "1247" |
| zone | iRODS zone | "iplant" |
| clientZone | iRODS client zone for proxy authentication | "iplant" |
| user | iRODS user id | "irods_user" |
| clientUser | iRODS client user id for proxy authentication | "irods_client_user" |
| password | iRODS user password | "password" in plain text |
| defaultResource | Default iRODS resource | "demoResc" |
| encryptionAlgorithm | iRODS encryption algorithm | "AES-256-CBC" |
| encryptionKeySize | iRODS encryption key size | "32" |
| encryptionSaltSize | iRODS encryption salt size | "8" |
| encryptionNumHashRounds | iRODS encryption hash rounds | "16" |
| caCertificateFile | TLS CA certificate file | "/etc/ssl/certs/ca.pem" |
| caCertificatePath | TLS CA certificate directory | "/etc/ssl/certs" |
| verifyServer | TLS server verification setting | "cert" |
| sslServerName | TLS server name | "data.cyverse.org" |
| path | iRODS collection path to mount. Required unless `pathMappings` is supplied. | "/iplant/home/irods_user" |
| pathMappings | JSON array of iRODS path mappings. Replaces `path` when supplied. | `[{"irods_path":"/iplant/home/user","mapping_path":"/","resource_type":"dir"}]` |
| readAheadMax | Maximum read-ahead size | "1048576" |
| uid | host system UID to map owner | -1 (executor's UID, mostly UID of root, 0) |
| gid | host system GID to map owner | -1 (executor's GID, mostly GID of root, 0) |
| systemUser | Host system user used by irodsfs | "root" |
| metadataConnection | JSON iRODS metadata connection configuration | `{}` |
| ioConnection | JSON iRODS I/O connection configuration | `{}` |
| cache | JSON iRODSFS cache configuration | `{}` |
| poolEndpoint | iRODSFS pool service endpoint | "tcp://irodsfs-pool.example.org:1247" |
| debug | Enable irodsfs debug logging | "true" |
| readOnly | Mount the volume read-only | "true" |
| volumeRootPath | iRODS path to mount. Creates a subdirectory per persistent volume. (only for dynamic volume provisioning) | "/iplant/home/irods_user" |
| noVolumeDir | "true" to not create a subdirectory under `volumeRootPath`. It mounts the `volumeRootPath`. (only for dynamic volume provisioning) | "false". "false" by default. |
| provisioningMode | Dynamic provisioning marker written by the driver. Do not set manually. | "dynamic" |
| enforceProxyAccess | "true" to mandate passing `clientUser`, or giving different `user` as in global configuration. | "false". "false" by default. |
| mountPathWhitelist | a comma-separated list of paths to allow mount. | "/iplant/home" |

### Install
#### Install with default configuration

Kubernetes > 1.17.x
```shell script
helm install irods-csi-driver .
```

Kubernetes < 1.17.x
```shell script
helm install irods-csi-driver --namespace kube-system .
```

#### Install with global configuration for proxy authentication
Edit `user_values.yaml` file for configuration.

Kubernetes > 1.17.x
```shell script
helm install irods-csi-driver -f user_values.yaml .
```

Kubernetes < 1.17.x
```shell script
helm install irods-csi-driver -f user_values.yaml --namespace kube-system .
```

#### Install in k0s
```shell script
helm install irods-csi-driver -f user_values.yaml --set kubeletDir=/var/lib/k0s/kubelet .
```

### Upgrade
```shell script
helm upgrade irods-csi-driver \
    --install . \
    --version 0.12.0 \
    -f values.yaml
```

### Uninstall
```shell script
helm uninstall irods-csi-driver
```
