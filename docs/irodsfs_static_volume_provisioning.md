# Create a Persistent Volume using iRODS CSI Driver

This document shows how to create a `Persistent Volume (PV)` using `static volume provisioning` in `iRODS CSI Driver`.

## Create a Storage Class (SC)

Define a `Storage Class` with following YAML file (`sc.yaml`).

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: irods-sc # the name of the SC
provisioner: irods.csi.cyverse.org
```

Create a Kubernetes object.

```shell script
kubectl apply -f sc.yaml
```

Check if the `Storage Class` is created successfully.

```shell script
kubectl get sc
```

The command will display following output.

```
NAME                   PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
irods-sc               irods.csi.cyverse.org   Delete          Immediate              false                  5s
local-path (default)   rancher.io/local-path   Delete          WaitForFirstConsumer   false                  7h
```

## Create a Secret

Define a `Secret` that stores access information with following YAML file (`secret.yaml`).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret # the name of the secret
type: Opaque
stringData:
  user: "my_username" # iRODS username
  password: "my_password" # iRODS password
```

Create a Kubernetes object.

```shell script
kubectl apply -f secret.yaml
```

Check if the `Secret` is created successfully.

```shell script
kubectl get secret
```

The command will display following output.

```
NAME        TYPE     DATA   AGE
my-secret   Opaque   2      40s
```

## Create a Persistent Volume (PV)

Define a `Persistent Volume` with following YAML file (`pv.yaml`).

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv # the name of the pv
  labels:
    vol-name: my-pv # same as the name
spec:
  capacity:
    storage: 5Gi # this is required but not meaningful (ignored by csi driver)
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: irods-sc # the name of Storage Class
  csi:
    driver: irods.csi.cyverse.org # the name of iRODS CSI driver
    volumeHandle: my-vol-id # unique volume ID
    volumeAttributes:
      client: "irodsfuse" # iRODS client
      host: "data.cyverse.org" # iRODS host
      port: "1247" # iRODS port
      zone: "iplant" # iRODS zone name
      path: "/iplant/home/my_username" # iRODS path to mount
    nodeStageSecretRef:
      name: "my-secret" # the name of the secret (read user and password from the secret)
      namespace: "default"
```

Create a Kubernetes object.

```shell script
kubectl apply -f pv.yaml
```

Check if the `Persistent Volume` is created successfully.

```shell script
kubectl get pv
```

The command will display following output.

```
NAME    CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS      CLAIM   STORAGECLASS   VOLUMEATTRIBUTESCLASS   REASON   AGE
my-pv   5Gi        RWX            Retain           Available           irods-sc       <unset>                          5s
```

## Create a Persistent Volume Claim (PVC)

Define a `Persistent Volume Claim` with following YAML file (`pvc.yaml`).

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc # the name of the pvc
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: irods-sc # the name of Storage Class
  selector:
    matchLabels:
      vol-name: my-pv # the name of the pv
  resources:
    requests:
      storage: 5Gi # this is required but not meaningful (must match to PV's storage capacity)
```

Create a Kubernetes object.

```shell script
kubectl apply -f pvc.yaml
```

Check if the `Persistent Volume Claim` is created successfully.

```shell script
kubectl get pvc
```

The command will display following output.

```
NAME     STATUS   VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS   VOLUMEATTRIBUTESCLASS   AGE
my-pvc   Bound    my-pv    5Gi        RWX            irods-sc       <unset>                 12s
```

## Use it in Apps

Mount the iRODS volume using the `Persistent Volume Claim` created above.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app # the name of the app
spec:
  containers:
  - name: app
    image: busybox
    command: ["/bin/sh"]
    args: ["-c", "echo Hello Kubernetes! $(date -u) >> /data/irods_csi_driver_out.txt"]
    volumeMounts:
    - name: persistent-storage # the name of the volume
      mountPath: /data # mount point
  restartPolicy: Never
  volumes:
  - name: persistent-storage # the name of the volume
    persistentVolumeClaim:
      claimName: my-pvc # the name of the PVC
```

The example app will create a file `irods_csi_driver_out.txt` at `/data` directory in the pod.
Because the `/data` directory is a mount point where `iRODS CSI Driver` mounts the iRODS path `/iplant/home/my_username` on. 
So this will create the file `/iplant/home/my_username/irods_csi_driver_out.txt`.