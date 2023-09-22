#! /usr/bin/env python3


### ======================================================================= ###
###     A command-line tool to dump irods csi driver logs                   ###
###     Uses: ./dump_irodscsidriver_logs.py                                 ###
### ======================================================================= ###

import os, sys
import argparse
import subprocess
from datetime import datetime

try:
    from subprocess import DEVNULL # py3k
except ImportError:
    import os
    DEVNULL = open(os.devnull, 'wb')

def get_kube_controller_pods(kubeconf=""):
    kubepods = []
    kubecommand = "kubectl get pods -n irods-csi-driver --no-headers --ignore-not-found -l app.kubernetes.io/instance=irods-csi-driver-controller --field-selector status.phase=Running -o name" + kubeconf
    
    pipe = os.popen(kubecommand)
    for line in pipe:
        podname = line.strip()
        if podname.startswith("pod/"):
            podname = podname[4:]

        kubepods.append(podname)

    return kubepods

def get_kube_node_pods(kubeconf=""):
    kubepods = []
    kubecommand = "kubectl get pods -n irods-csi-driver --no-headers --ignore-not-found -l app.kubernetes.io/instance=irods-csi-driver-node --field-selector status.phase=Running -o name" + kubeconf
    
    pipe = os.popen(kubecommand)
    for line in pipe:
        podname = line.strip()
        if podname.startswith("pod/"):
            podname = podname[4:]

        kubepods.append(podname)            

    return kubepods

def dump_controllers(kubeconf, dumpdir):
    pods = get_kube_controller_pods(kubeconf)

    for pod in pods:
        for container in ["irods-plugin", "csi-provisioner"]:
            container_flag = " -c " + container

            log_filename = dumpdir + "/" + pod + "_" + container + ".log"
            with open(log_filename, "w") as logfile:
                kubecommand = "kubectl logs -n irods-csi-driver" + container_flag + kubeconf + " " + pod
                p = subprocess.Popen(kubecommand, shell=True, universal_newlines=True, stdout=logfile)
                p.wait()

def dump_nodes(kubeconf, dumpdir):
    pods = get_kube_node_pods(kubeconf)

    for pod in pods:
        for container in ["irods-plugin", "csi-driver-registrar", "irods-pool"]:
            container_flag = " -c " + container

            log_filename = dumpdir + "/" + pod + "_" + container + ".log"
            with open(log_filename, "w") as logfile:
                kubecommand = "kubectl logs -n irods-csi-driver" + container_flag + kubeconf + " " + pod
                p = subprocess.Popen(kubecommand, shell=True, universal_newlines=True, stdout=logfile)
                p.wait()

        # dump irodsfs /storage/irodsfs
        irodsfs_dumpdir = dumpdir + "/" + pod
        if not os.path.exists(irodsfs_dumpdir):
            os.makedirs(irodsfs_dumpdir)

        kubecommand = "kubectl cp -c irods-plugin" + kubeconf + " irods-csi-driver/" + pod + ":/storage/irodsfs " + irodsfs_dumpdir + "/"
        p = subprocess.Popen(kubecommand, shell=True, universal_newlines=True, stdout=DEVNULL, stderr=DEVNULL)
        p.wait()


parser = argparse.ArgumentParser()
parser.add_argument("--kubeconfig", dest="kubeconfig", type=str, help="kubernetes configuration filepath")
parser.add_argument("-o", dest="output", type=str, help="dump output directory")

args = parser.parse_args()

kubeconf = ""
if args.kubeconfig:
    kubeconf = " --kubeconfig=" + args.kubeconfig

now = datetime.now()
dumpdir = now.strftime("irodscsi_log_%Y%m%d_%H%M%S")

if args.output:
    dumpdir = args.output

if not os.path.exists(dumpdir):
    os.makedirs(dumpdir)

dump_controllers(kubeconf, dumpdir)
dump_nodes(kubeconf, dumpdir)
