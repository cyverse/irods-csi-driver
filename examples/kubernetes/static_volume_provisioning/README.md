## Static Volume Provisioning

In static volume provisioning, persistent volumes must be created before they
are claimed. Each example includes a `pv.yaml` with the connection settings in
`spec.csi.volumeAttributes`.

The CSI node plugin sends the selected client configuration to the host's
`irodsfsd` service. Install and start `irodsfsd` on every node where a pod may
mount one of these volumes.

## iRODS Configuration

The `volumeAttributes` in `pv.yaml` contain non-sensitive connection settings.
The `irodsfuse_secrets` example demonstrates passing credentials with
`nodeStageSecretRef`; the basic examples retain inline placeholder credentials
only for minimal manual testing.

### iRODS Client Configuration

The following client types are supported. `irodsfsd` performs the actual
filesystem-specific mount on the host.

| client | Host service requirement |
| --- | --- |
| `irodsfuse` | iRODSFS support in irodsfsd |
| `webdav` | A compatible WebDAV server, such as iRODS-WebDAV or Davrods |
| `nfs` | An accessible NFS export |

#### iRODS FUSE Client
| Field | Description | Example |
| --- | --- | --- |
| client | Client type | "irodsfuse" |
| authenticationScheme | iRODS authentication scheme | "native" |
| clientServerNegotiation | iRODS client-server negotiation setting | "request_server_negotiation" |
| clientServerPolicy | iRODS client-server negotiation policy | "CS_NEG_REQUIRE" |
| host | iRODS hostname | "data.cyverse.org" |
| port | iRODS port | Optional, Default "1247" |
| zone | iRODS zone | "iplant" |
| user | iRODS user id | "irods_user" |
| password | iRODS user password | supplied through `nodeStageSecretRef` |
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
| path | iRODS collection path to mount. Required unless `pathMappings` is supplied. | "/iplant/home/irods_user" |
| pathMappings | JSON array of iRODS path mappings. Replaces `path` when supplied. | `[{"irods_path":"/iplant/home/user","mapping_path":"/","resource_type":"dir"}]` |
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

Use canonical lowerCamelCase field names exactly as shown. Other spellings are
ignored.

#### WebDAV Client
| Field | Description | Example |
| --- | --- | --- |
| client | Client type | "webdav" |
| user | WebDAV user name, or omit for anonymous access | "user" |
| password | WebDAV password, or omit for anonymous access | supplied through `nodeStageSecretRef` |
| url | URL | "https://data.cyverse.org/dav/iplant/home/irods_user" |
| config | Additional DAVFS configuration as comma-separated key-value pairs | "key1=value1,key2=value2" |
| readOnly | Mount the volume read-only | "true" |

Mounts **url**

#### NFS Client
| Field | Description | Example |
| --- | --- | --- |
| client | Driver type | "nfs" |
| host | NFS hostname | "nfs.example.org" |
| port | NFS port | Optional, defaults to "2049" |
| path | NFS export path | "/exports/data" |
| readOnly | Mount the volume read-only | "true" |

Mounts **host**:/**path**

### Kubernetes Secrets

Kubernetes Secrets can be used to pass credentials through
`nodeStageSecretRef`. The mounted CSI driver secret provides global defaults,
then `volumeAttributes` override the node-stage secret, and driver defaults
have the highest priority. In other words:

```text
driver global secret > PV volumeAttributes > nodeStageSecretRef secret
```

Put a common iRODSFS `poolEndpoint` in the driver global secret to apply it to
all iRODSFS static volumes. Put it in `volumeAttributes` when it must vary by
PV and no global endpoint is configured.

The `irodsfuse_proxyauth` example demonstrates proxy authentication using a
driver global Secret for the proxy identity and PV `volumeAttributes` for the
client user and iRODS path.

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
