## iRODS CSI Driver Helm Chart
This script enables easy installation of iRODS CSI Driver using Helm Chart.

## Prerequisites
- Helm 3+
- Kubernetes > 1.17.x, can be deployed to any namespace.
- Kubernetes < 1.17.x, namespace **must** be `kube-system`, as `system-cluster-critical` hard coded to this namespace.

## Install the chart
```shell script
helm install irods-csi-driver .
```

## Upgrade release
```shell script
helm upgrade irods-csi-driver \
    --install . \
    --version 0.2.0 \
    --namespace kube-system \
    -f values.yaml
```

## Uninstalling the chart
```shell script
helm delete irods-csi-driver
```
