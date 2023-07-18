#! /usr/bin/python

### ======================================================================= ###
###     A Nagios plugin to check irods csi driver mounts                    ###
###     Uses: ./check_irodscsidriver_mounts.py                              ###
### ======================================================================= ###

import os, sys
import argparse

def get_mounts_status():
    mounts = []
    with open("/proc/mounts", "r") as f:    
        for line in f:
            fields = line.strip().split()
            if len(fields) < 3:
                continue

            progname = fields[0].strip()

            if progname.lower() in ["irodsfs"]:
                mounts.append(line)
    
    return mounts

def check_irodsfs_mounts():
    mounts = get_mounts_status()

    running_mounts = []
    failed_mounts = []

    for line in mounts:
        fields = line.strip().split()
        if len(fields) < 3:
            continue
        
        mountpoint = fields[1].strip()

        try:
            # Check if the mount point exists and is a directory
            if os.path.exists(mountpoint) and os.path.isdir(mountpoint) and os.path.ismount(mountpoint):
                #valid
                running_mounts.append(mountpoint)
            else:
                failed_mounts.append(mountpoint)
        except Exception as e:
            failed_mounts.append(mountpoint)

    if len(failed_mounts) == 0:
        return 0, "OK - iRODS CSI Driver is running well."

    print_mounts = ', '.join(failed_mounts)
    return 2, "CRITICAL - Found failed iRODS CSI Driver mount points at [%s]." % print_mounts


error_code, msg = check_irodsfs_mounts()
print(msg)
sys.exit(error_code)
