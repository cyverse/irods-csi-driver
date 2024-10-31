# Install iRODS CSI Driver using Helm

You will need [Helm](https://helm.sh/docs/helm/helm_install/) to install `iRODS CSI Driver`.

## Add Helm Chart Repository to Helm

Run following command to add `iRODS CSI Driver Helm Chart Repository`.
```shell script
helm repo add irods-csi-driver-repo https://cyverse.github.io/irods-csi-driver-helm/
helm repo update
```

Verify if the repository is added successfully.
```shell script
helm search repo irods
```

## Install iRODS CSI Driver

Install `iRODS CSI Driver` with default configurations.
The command below will install `irods-csi-driver-repo/irods-csi-driver` chart and the installed driver will be named `irods-csi-driver`. The driver pods will be created in `irods-csi-driver` namespace.

```shell script
helm install --create-namespace -n irods-csi-driver irods-csi-driver irods-csi-driver-repo/irods-csi-driver
```

Check pods of `iRODS CSI Driver`.

```shell script
kubectl get pods -n irods-csi-driver
```

The command will display following output.
```
NAME                                           READY   STATUS    RESTARTS   AGE
irods-csi-driver-controller-6c7bb75479-d7z4p   2/2     Running   0          35m
irods-csi-driver-controller-6c7bb75479-nk6cd   2/2     Running   0          35m
irods-csi-driver-node-zbnkp                    4/4     Running   0          35m
```

By default, `iRODS CSI Driver` will create:
- two `irods-csi-driver-controller` pods in a cluster
- one `irods-csi-driver-node` pod per cluster node

## Advanced configuration

### Cache Configuration

`iRODS FUSE Lite Pool Server` is built-in `iRODS CSI Driver` to provide connection pooling and data caching. The server runs in `irods-csi-driver-node` pod.

To configure cache-related settings, create a YAML file that adds `nodeService/irodsPool/extraArgs`. 

For example, the following sets `cache timeout` for paths, increase `max cache size`, and set `data root path` for storing cache and log.

```yaml
nodeService:
  irodsPool:
    extraArgs:
      - '--cache_timeout_settings=[{"path":"/","timeout":"-1ns","inherit":false},{"path":"/cyverse","timeout":"-1ns","inherit":false},{"path":"/cyverse/home","timeout":"1h","inherit":false},{"path":"/cyverse/home/shared","timeout":"1h","inherit":true}]'
      - --cache_size_max=10737418240
      - --data_root=/irodsfs-pool
```

Then, provide the YAML file when installing `iRODS CSI Driver` using Helm.

```shell script
helm install --create-namespace -n irods-csi-driver irods-csi-driver irods-csi-driver-repo/irods-csi-driver -f ./pool_config.yaml
```

### Volume Configuration

To configure default volume settings, create a YAML file that adds `globalConfig/secret/stringData`. 

For example, the following sets default `client`, `host`, `port`, `zone`, `user`, `password` for iRODS access.

Set `retainData` to `false` to delete the volume directory in iRODS after use (only for dynamic volume provisioning mode).
Set `enforceProxyAccess` to `true` for only allowing proxy access to iRODS.
Set `mountPathWhitelist` to allow mounting certain iRODS paths.

```yaml
globalConfig:
  secret:
    stringData:
      client: "irodsfuse"
      host: "bishop.cyverse.org"
      port: "1247"
      zone: "cyverse"
      user: "de-irods"
      retainData: "false"
      password: "real-password-here"
      enforceProxyAccess: "true"
      mountPathWhitelist: "/cyverse/home"
```

Then, provide the YAML file when installing `iRODS CSI Driver` using Helm.

```shell script
helm install --create-namespace -n irods-csi-driver irods-csi-driver irods-csi-driver-repo/irods-csi-driver -f ./volume_config.yaml
```


