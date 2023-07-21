#! /usr/bin/python

### ======================================================================= ###
###     A Nagios plugin to check irods csi driver pods                      ###
###     Uses: ./check_irodscsidriver_pods.py                                ###
### ======================================================================= ###

import os, sys
import socket
import argparse

def get_hostnames():
    hostnames = []
    addrs = socket.gethostbyaddr(socket.gethostname())

    hostnames.append(addrs[0]) # hostname
    for addrfield in addrs[1:]: # alias and ip addresses
        # addrfield is an array
        for v in addrfield:
            hostnames.append(v) # alias
    
    return hostnames

def get_hostnames_for(hostname):
    hostnames = []
    hostnames.append(hostname)

    fqdn = socket.getfqdn(hostname)
    if hostname != fqdn:
        hostnames.append(fqdn)

    return hostnames

def get_kube_pods_status(hostnames):
    kubepods = []
    for hostname in hostnames:
        kubecommand = "kubectl get pods -n irods-csi-driver -o wide --no-headers --ignore-not-found -l app.kubernetes.io/instance=irods-csi-driver-node --field-selector spec.nodeName=%s" % hostname
        kubecommand = kubecommand + kubeconf

        pipe = os.popen(kubecommand)
        for line in pipe:
            kubepods.append(line)
    
    return kubepods

def check_kube_pods(hostnames):
    kubepods = get_kube_pods_status(hostnames)

    running_pods = []
    restarted_pods = []
    restarted_toomany_pods = []
    stopped_pods = []

    for line in kubepods:
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

        if status.lower() in ["running"]:
            running_pods.append(msg)
        else:
            stopped_pods.append(msg)
            continue

        if restarts > 0:
            restarted_pods.append(msg)

            if restarts > 10:
                restarted_toomany_pods.append(msg)
            continue


    if len(running_pods) == 1 and len(restarted_pods) == 0 and len(stopped_pods) == 0:
        return 0, "OK - iRODS CSI Driver is running well."
    elif len(running_pods) == 0:
        return 2, "CRITICAL - iRODS CSI Drivers are not running. No pods is running."
    elif len(running_pods) > 1:
        print_pods = ', '.join(running_pods)
        return 2, "CRITICAL - iRODS CSI Drivers are running, but more than 1. Running pods are [%s]." % print_pods
    elif len(stopped_pods) > 0:
        print_pods = ', '.join(stopped_pods)
        return 2, "CRITICAL - iRODS CSI Drivers are not running. Failed pods are [%s]." % print_pods
    elif len(restarted_toomany_pods) > 0:
        print_pods = ', '.join(restarted_toomany_pods)
        return 2, "CRITICAL - iRODS CSI Drivers are restarted more than 10 times. Restarted pods are [%s]. Check irodsfs mounts." % print_pods
    elif len(restarted_pods) > 0:
        print_pods = ', '.join(restarted_pods)
        return 1, "WARNING - iRODS CSI Drivers are restarted. Restarted pods are [%s]. Check irodsfs mounts." % print_pods
    else:
        return 3, "UKNOWN - iRODS CSI Driver status unknown"


parser = argparse.ArgumentParser()
parser.add_argument("--hostname", dest="hostname", type=str, help="current node's hostname")
parser.add_argument("--kubeconfig", dest="kubeconfig", type=str, help="kubernetes configuration filepath")

args = parser.parse_args()

kubeconf = ""
if args.kubeconfig:
    kubeconf = " --kubeconfig=" + args.kubeconfig

hostnames = []

if args.hostname:
    hostnames = get_hostnames_for(args.hostname)
else:
    hostnames = get_hostnames()

if len(hostnames) == 0:
    print("UKNOWN - Hostname not given")
    sys.exit(3)

error_code, msg = check_kube_pods(hostnames)
print(msg)
sys.exit(error_code)
