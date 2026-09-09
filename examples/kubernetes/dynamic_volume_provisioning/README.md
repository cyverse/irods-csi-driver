## Dynamic Volume Provisioning

In dynamic volume provisioning, the CSI controller creates an iRODS directory
when a PersistentVolumeClaim is provisioned. The StorageClass parameters are
stored in the generated PV's `volumeAttributes` and are later used by the CSI
node plugin to mount the volume through the host's `irodsfsd` service.

Only `irodsfuse` is supported for dynamic provisioning. Volume deletion
releases the Kubernetes volume reference but always retains the iRODS data.

## iRODS Configuration

Store non-sensitive mount settings in `storageclass.yaml`. Use Kubernetes
Secrets for credentials.

### iRODS Client Configuration

Dynamic provisioning uses `irodsfuse`; `irodsfsd` on the selected node creates
the filesystem mount.

#### iRODS FUSE Client
| Field | Description | Example |
| --- | --- | --- |
| client | Client type. Must be `irodsfuse`. | "irodsfuse" |
| authenticationScheme | iRODS authentication scheme | "native" |
| clientServerNegotiation | iRODS client-server negotiation setting | "request_server_negotiation" |
| clientServerPolicy | iRODS client-server negotiation policy | "CS_NEG_REQUIRE" |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional, Default "1247" |
| zone | iRODS zone | "iplant" |
| user | iRODS user id used by the controller. Required; anonymous users cannot dynamically provision. | "irods_user" |
| password | iRODS user password | supplied by `Secrets` |
| clientZone | iRODS client zone for proxy authentication | "iplant" |
| clientUser | iRODS client user id for proxy authentication | "irods_client_user" |

| defaultResource | Default iRODS resource | "demoResc" |
| encryptionAlgorithm | iRODS encryption algorithm | "AES-256-CBC" |
| encryptionKeySize | iRODS encryption key size | "32" |
| encryptionSaltSize | iRODS encryption salt size | "8" |
| encryptionNumHashRounds | iRODS encryption hash rounds | "16" |
| caCertificateFile | TLS CA certificate file | "/etc/ssl/certs/ca.pem" |
| caCertificatePath | TLS CA certificate directory | "/etc/ssl/certs" |
| verifyServer | TLS server verification setting | "cert" |
| sslServerName | TLS server name | "data.cyverse.org" |
| volumeRootPath | Required iRODS root path. A directory named after the CSI volume is created beneath it. | "/iplant/home/irods_user" |
| noVolumeDir | Use `volumeRootPath` itself instead of creating a per-volume directory. | "false" |
| readAheadMax | Maximum read-ahead size | "1048576" |
| uid | Host system UID | "1000" |
| gid | Host system GID | "1000" |
| systemUser | Host system user used by irodsfs | "root" |
| metadataConnection | JSON iRODS metadata connection configuration | `{}` |
| ioConnection | JSON iRODS I/O connection configuration | `{}` |
| cache | JSON iRODSFS cache configuration | `{}` |
| poolEndpoint | iRODSFS pool service endpoint | "tcp://irodsfs-pool.example.org:1247" |
| debug | Enable irodsfs debug logging | "true" |
| readOnly | Mount the volume read-only | "true" |
| enforceProxyAccess | Require proxy authentication | "true" |
| mountPathWhitelist | Comma-separated iRODS paths allowed to mount | "/iplant/home" |

`path` and `pathMappings` are assigned by the controller for dynamically
provisioned volumes; do not set them in the StorageClass. Use canonical
lowerCamelCase parameter names exactly as shown. Other spellings are ignored.

### Kubernetes Secrets

Use `csiProvisionerSecretName` and `csiProvisionerSecretNamespace` when the
controller needs credentials to create the iRODS directory. Use
`csiNodeStageSecretName` and `csiNodeStageSecretNamespace` when the node needs
credentials to mount the volume. The same Secret may be used for both.

At mount time, configuration precedence is:

```text
driver global secret > StorageClass parameters > node-stage Secret
```

Put a common iRODSFS `poolEndpoint` in the driver global secret to apply it to
all iRODSFS volumes. Put it in StorageClass parameters only when no
global endpoint is configured and the endpoint should be specific to that
StorageClass.

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
