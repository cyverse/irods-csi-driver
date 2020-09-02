
## iRODS CSI Driver Helm Chart
For more info view [irods-csi-driver](https://github.com/cyverse/irods-csi-driver)

## Prerequisites
- Helm 3+
- Kubernetes > 1.17.x, can be deployed to any namespace.
- Kubernetes < 1.17.x, namespace **must** be `kube-system`, as `system-cluster-critical` hard coded to this namespace.

## Install chart
```shell script
helm install . --name irods-csi-driver
```

## Upgrade release
```shell script
helm upgrade irods-csi-driver \
    --install . \
    --version 0.2.0 \
    --namespace kube-system \
    -f values.yaml
```

## Uninstalling the Chart
```shell script
helm delete irods-csi-driver --namespace [NAMESPACE]
```

