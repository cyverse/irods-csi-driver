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

### irodsfsd Endpoint

Install and start `irodsfsd` as a host service on every node that can run a
CSI node or controller pod. The default endpoint is `tcp://127.0.0.1:13020`.
To use a different endpoint, configure both plugin values:

```yaml
controllerService:
  irodsPlugin:
    irodsfsdEndpoint: tcp://127.0.0.1:13020
nodeService:
  irodsPlugin:
    irodsfsdEndpoint: tcp://127.0.0.1:13020
```

`irodsfsd` owns filesystem-specific clients, connection pooling, and cache
configuration. The CSI deployment does not run an `irodsfsd` or pool sidecar.

### Volume Configuration

To configure default volume settings, create a YAML file that adds `globalConfig/secret/stringData`. 

For example, the following sets default `client`, `host`, `port`, `zone`, `user`, `password` for iRODS access.

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
      password: "real-password-here"
      enforceProxyAccess: "true"
      mountPathWhitelist: "/cyverse/home"
```

Then, provide the YAML file when installing `iRODS CSI Driver` using Helm.

```shell script
helm install --create-namespace -n irods-csi-driver irods-csi-driver irods-csi-driver-repo/irods-csi-driver -f ./volume_config.yaml
```
