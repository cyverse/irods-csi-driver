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

def get_kube_pods_status(hostnames, kubeconf=""):
    kubepods = []
    kubecommand = "kubectl get pods -n irods-csi-driver -o wide --no-headers --ignore-not-found -l app.kubernetes.io/instance=irods-csi-driver-node" + kubeconf
    
    pipe = os.popen(kubecommand)
    for line in pipe:
        fields = line.strip().split()
        if len(fields) < 9:
            continue

        podname = fields[0].strip()
        status = fields[2].strip()
        restarts = int(fields[3].strip())
        ip = fields[5].strip()
        node = fields[6].strip()
        msg = "%s(%s)" % (podname, node)
        
        if fields[4].startswith("("):
            for i in range(4, 7):
                if fields[i].strip().endswith(")"):
                    ip = fields[i+2].strip()
                    node = fields[i+3].strip()
                    msg = "%s(%s)" % (podname, node)

        match = False
        for hostname in hostnames:
            if ip == hostname:
                # match
                match = True
                break
            elif node == hostname:
                match = True
                break
        
        if match:
            kubepods.append((podname, status, restarts, ip, node, msg))

    return kubepods

def check_kube_pods(hostnames, restart_critical, restart_warning, kubeconf=""):
    kubepods = get_kube_pods_status(hostnames, kubeconf)

    running_pods = []
    warning_pods = []
    critical_pods = []
    stopped_pods = []

    for pod in kubepods:
        _, status, restarts, ip, node, msg = pod

        if status.lower() in ["running"]:
            running_pods.append(msg)

            if restarts > restart_critical:
                critical_pods.append(msg)
            elif restarts > restart_warning:
                warning_pods.append(msg)
        else:
            stopped_pods.append(msg)


    if len(running_pods) == 1 and len(warning_pods) == 0 and len(critical_pods) == 0 and len(stopped_pods) == 0:
        return 0, "OK - iRODS CSI Driver is running well"
    elif len(running_pods) == 0:
        return 2, "CRITICAL - iRODS CSI Drivers are not running. No pods is running"
    elif len(running_pods) > 1:
        print_pods = ', '.join(running_pods)
        return 2, "CRITICAL - iRODS CSI Drivers are running, but more than 1. Running pods are [%s]" % print_pods
    elif len(stopped_pods) > 0:
        print_pods = ', '.join(stopped_pods)
        return 2, "CRITICAL - iRODS CSI Drivers are not running. Failed pods are [%s]" % print_pods
    elif len(critical_pods) > 0:
        print_pods = ', '.join(critical_pods)
        return 2, "CRITICAL - iRODS CSI Drivers are restarted more than 10 times. Restarted pods are [%s]" % print_pods
    elif len(warning_pods) > 0:
        print_pods = ', '.join(warning_pods)
        return 1, "WARNING - iRODS CSI Drivers are restarted. Restarted pods are [%s]" % print_pods
    else:
        return 3, "UKNOWN - iRODS CSI Driver status unknown"


parser = argparse.ArgumentParser()
parser.add_argument("--hostname", dest="hostname", type=str, help="current node's hostname")
parser.add_argument("--kubeconfig", dest="kubeconfig", type=str, help="kubernetes configuration filepath")
parser.add_argument("--restart_critical", dest="restart_critical", type=int, default=10, help="set the number of pod restarts to be considered critical")
parser.add_argument("--restart_warning", dest="restart_warning", type=int, default=3, help="set the number of pod restarts to be considered warning")

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

restart_critical = 10
if args.restart_critical:
    restart_critical = args.restart_critical

restart_warning = 3
if args.restart_warning:
    restart_warning = args.restart_warning

error_code, msg = check_kube_pods(hostnames, restart_critical, restart_warning, kubeconf)
print(msg)
sys.exit(error_code)
