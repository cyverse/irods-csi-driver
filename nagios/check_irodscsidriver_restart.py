#! /usr/bin/python


### ======================================================================= ###
###     A Nagios plugin to check irods csi driver restart                   ###
###     Uses: ./check_irodscsidriver_restart.py                            ###
### ======================================================================= ###

import os, sys

hostname = os.getenv("HOSTNAME")
kubecommand = "kubectl get pods -n irods-csi-driver -o wide --no-headers --field-selector spec.nodeName=%s" % hostname
kubeconf = ""
if len(sys.argv) >= 2:
    kubeconf = sys.argv[1]

if len(kubeconf) > 0:
    kubecommand = kubecommand + " --kubeconfig=" + kubeconf

pipe = os.popen(kubecommand)

restarted_pods = []
restarted_toomany_pods = []
stopped_pods = []

for line in pipe:
    fields = line.strip().split()
    if len(fields) < 9:
        continue
    
    podname = fields[0].strip()
    status = fields[2].strip()
    restarts = int(fields[3].strip())
    msg = "%s(%s)" % (podname, fields[6].strip())

    if fields[4].startswith("("):
        for i in range(4, 7):
            if fields[i].strip().endswith(")"):
                msg = "%s(%s)" % (podname, fields[i+3].strip())

    if not podname.startswith("irods-csi-driver-node"):
        continue

    if status.lower() not in ["running"]:
        stopped_pods.append(msg)
        continue

    if restarts > 0:
        restarted_pods.append(msg)

        if restarts > 10:
            restarted_toomany_pods.append(msg)
        continue
    

if len(restarted_pods) == 0 and len(stopped_pods) == 0:
    print("OK - iRODS CSI Driver is running well.")
    sys.exit(0)
elif len(stopped_pods) > 0:
    print_pods = ', '.join(stopped_pods)
    print("CRITICAL - iRODS CSI Drivers are not running. Failed pods are [%s]." % print_pods)
    sys.exit(2)
elif len(restarted_toomany_pods) > 0:
    print_pods = ', '.join(restarted_toomany_pods)
    print("CRITICAL - iRODS CSI Drivers are restarted more than 10 times. Restarted pods are [%s]. Check irodsfs mounts." % print_pods)
    sys.exit(2)
elif len(restarted_pods) > 0:
    print_pods = ', '.join(restarted_pods)
    print("WARNING - iRODS CSI Drivers are restarted. Restarted pods are [%s]. Check irodsfs mounts." % print_pods)
    sys.exit(1)
else:
    print("UKNOWN - iRODS CSI Driver status unknown")
    sys.exit(3)
