## iRODS CSI Driver Helm Chart
This script enables easy installation of iRODS CSI Driver using Helm Chart.

### Compatibility
- Helm 3+
- Kubernetes > 1.17.x, can be deployed to any namespace.
- Kubernetes < 1.17.x, namespace **must** be `kube-system`, as `system-cluster-critical` hard coded to this namespace.

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
helm install irods-csi-driver -f user_values.yaml --set kubletDir=/var/lib/k0s/kubelet .
```

### Upgrade
```shell script
helm upgrade irods-csi-driver \
    --install . \
    --version 0.11.4 \
    -f values.yaml
```

### Uninstall
```shell script
helm uninstall irods-csi-driver
```

