#! /usr/bin/python


### ======================================================================= ###
###     A Nagios plugin to check irods csi driver restart                   ###
###     Uses: ./check_irodscsidriver_restart.py                            ###
### ======================================================================= ###

import os, sys

kubecommand = "kubectl get pods -n irods-csi-driver -o wide --no-headers"
kubeconf = ""
if len(sys.argv) >= 2:
    kubeconf = sys.argv[1]

if len(kubeconf) > 0:
    kubecommand = kubecommand + " --kubeconfig=" + kubeconf

pipe = os.popen(kubecommand)

restarted_pods = []
stopped_pods = []

for line in pipe:
    fields = line.strip().split()
    if len(fields) < 9:
        continue
    
    podname = fields[0].strip()
    status = fields[2].strip()
    restarts = int(fields[3].strip())
    node = "%s(%s)" % (fields[6].strip(), fields[5].strip())

    if fields[4].startswith("("):
        for i in range(4, 7):
            if fields[i].strip().endswith(")"):
                node = "%s(%s)" % (fields[i+3].strip(), fields[i+2].strip())

    if not podname.startswith("irods-csi-driver-node"):
        continue

    if restarts > 0:
        restarted_pods.append(node)
        continue

    if status.lower() not in ["running"]:
        stopped_pods.append(node)
        continue

if len(restarted_pods) == 0 and len(stopped_pods) == 0:
    print("OK - iRODS CSI Driver is running well.")
    sys.exit(0)
elif len(stopped_pods) > 0:
    print_pods = ', '.join(stopped_pods)
    print("CRITICAL - iRODS CSI Drivers are not running on [%s] nodes." % print_pods)
    sys.exit(2)
elif len(restarted_pods) > 0:
    print_pods = ', '.join(restarted_pods)
    print("WARNING - iRODS CSI Drivers are restarted on [%s] nodes. Check irodsfs mounts." % print_pods)
    sys.exit(1)
else:
    print("UKNOWN - iRODS CSI Driver status unknown")
    sys.exit(3)
