## iRODS CSI Driver Installation on Kubernetes
This directory contains YAML files to install iRODS CSI Driver on Kubernetes.

## Prerequisites

Install and start `irodsfsd` as a system service on every node that runs the
CSI node pod. The base node manifest uses host networking and connects to
`tcp://127.0.0.1:13020`; change `IRODSFSD_ENDPOINT` if the service uses a
different endpoint. The controller pod does not connect to `irodsfsd`.

The CSI node plugin keeps bidirectional mount propagation so mounts created by
the host daemon are visible to kubelet and pods. Do not add an `irodsfsd` or
`irods-pool` sidecar to this deployment.

The manifests support Linux AMD64 and ARM64 nodes. Publish or configure an
ARM64-compatible driver image before deploying to ARM nodes; the Kubernetes
CSI sidecar images used here are multi-architecture. `irodsfsd` on each ARM64
node must also be an ARM64-compatible build.

## Install the driver
Install the stable driver:

```shell script
kubectl apply -k "overlays/stable"
```

Install the development driver:
```shell script
kubectl apply -k "overlays/dev"
```

Verify the driver installation:
```shell script
kubectl get csinodes -o jsonpath='{range .items[*]} {.metadata.name}{": "} {range .spec.drivers[*]} {.name}{"\n"} {end}{end}'
```

## Uninstall the driver
Uninstall the stable driver:
```shell script
kubectl delete -k "overlays/stable"
```

Uninstall the development driver:
```shell script
kubectl delete -k "overlays/dev"
```
