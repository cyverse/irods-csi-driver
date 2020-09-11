## iRODS CSI Driver Helm Chart
This script enables easy installation of iRODS CSI Driver using Helm Chart.

### Compatibility
- Helm 3+
- Kubernetes > 1.17.x, can be deployed to any namespace.
- Kubernetes < 1.17.x, namespace **must** be `kube-system`, as `system-cluster-critical` hard coded to this namespace.

### Install
#### Install with default configuration
```shell script
helm install irods-csi-driver .
```

#### Install with global configuration for proxy authentication
Edit `user_values.yaml` file for configuration.

```shell script
helm install irods-csi-driver -f user_values.yaml .
```

### Upgrade
```shell script
helm upgrade irods-csi-driver \
    --install . \
    --version 0.2.1 \
    --namespace kube-system \
    -f values.yaml
```

### Uninstall
```shell script
helm delete irods-csi-driver
```
