#! /usr/bin/env python3

import os
import sys
import getpass

from irods.session import iRODSSession
from irods.exception import CollectionDoesNotExist

# irods_ls.py hostname:port zone /parentdir/targetdir
# iRODS Username and Password are passed via STDIN

def main(argv):
    if len(argv) < 4:
        print("Arguments not given correctly (given = %d)" % len(argv), file=sys.stderr)
        sys.exit(1)

    hostport = argv[1]
    zone = argv[2]
    path = argv[3]

    host = hostport
    port = 1247

    if ":" in hostport:
        hostport_vars = hostport.split(":")
        host = hostport_vars[0].strip()
        port = int(hostport_vars[1].strip())

    if sys.stdin.isatty():
        user = input("Username: ")
        password = getpass.getpass("Password: ")
        clientUser = input("Client Username: ")
    else:
        user = sys.stdin.readline().rstrip()
        password = sys.stdin.readline().rstrip()
        clientUser = sys.stdin.readline().rstrip()

    if not host:
        print("iRODS HOST is not given", file=sys.stderr)
        sys.exit(1)

    if not zone:
        print("iRODS ZONE is not given", file=sys.stderr)
        sys.exit(1)

    if port <= 0:
        port = 1247

    if not user:
        print("iRODS USER is not given", file=sys.stderr)
        sys.exit(1)

    if not password:
        print("iRODS PASSWORD is not given", file=sys.stderr)
        sys.exit(1)

    if not path:
        print("iRODS PATH is not given", file=sys.stderr)
        sys.exit(1)

    zonepath = path
    if not path.startswith("/" + zone + "/"):
        zonepath = "/" + zone + "/" + path.lstrip("/")

    if len(clientUser) == 0:
        clientUser = None

    with iRODSSession(host=host, port=port, user=user, password=password, zone=zone, client_user=clientUser) as session:
        try:
            coll = session.collections.get(zonepath)
            if coll:
                for obj in coll.data_objects:
                    print(obj, file=sys.stdout)

                for col in coll.subcollections:
                    print(col, file=sys.stdout)
                
                sys.exit(0)
        except CollectionDoesNotExist:
            print("Could not list a path %s" % zonepath, file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main(sys.argv)
